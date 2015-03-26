#define _BSD_SOURCE

#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <getopt.h>
#include <stdarg.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <syslog.h>
#include <sys/poll.h>
#include <unistd.h>

#include <libwebsockets.h>
#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/ssl.h>

#define LOG_IDENT "lua-ws-wrapper"

enum {
	ERR_NONE = 0,
	ERR_PROTOCOL_VIOLATION = 1,
	ERR_NETWORK = 2,
	ERR_AUTHENTICATION_FAILED = 3,
	ERR_ARG = 4,
	ERR_PIPE = 5,

	ERR_INTERNAL = 64,
};

static const char* host;
static unsigned port;
static const char* user;
static const char* device;
static const char* caPath;
static char wsPath[4096];
static unsigned char devKey[128], devKeyLen;
static bool forked, forkWhenReady;

static bool hexDecode(const char* in, unsigned char* out, int len)
{
	for (int i = 0; i < len; i++) {
		if (!isxdigit(in[2 * i]) || !isxdigit(in[2 * i + 1]))
			return false;

		sscanf(in + 2 * i, "%2hhx", &out[i]);
	}
	return true;
}

static void hexEncode(const unsigned char* in, char* out, int len)
{
	while (len-- > 0) {
		sprintf(out, "%02x", *in);
		in++;
		out += 2;
	}
}

static void ws_log(const char* fmt, ...)
{
	va_list args;

	va_start(args, fmt);
	if (!forked) {
		vfprintf(stderr, fmt, args);
		fprintf(stderr, "\n");
	} else {
		vsyslog(LOG_ERR, fmt, args);
	}
	va_end(args);
}

enum {
	CS_AUTH_INIT,
	CS_AUTH_COMPUTED,
	CS_AUTH_SENT,
	CS_BRIDGING,
};

static int clientState = CS_AUTH_INIT;
static unsigned char response[32];
static struct pollfd pollSet[2];
static char inputBuffer[4096];
static unsigned inputBufferLength;
static const char authResponseOK[] = "proceed";

void go_to_background()
{
	switch (fork()) {
	case 0:
		forked = true;
		openlog(LOG_IDENT, 0, LOG_DAEMON);
		lws_set_log_level(LLL_ERR | LLL_WARN, lwsl_emit_syslog);
		fclose(stderr);
		break;

	case -1:
		ws_log("could not fork");
		exit(ERR_INTERNAL);

	default:
		exit(ERR_NONE);
	}
}

static int msg_1_device_callback(struct libwebsocket_context *context, struct libwebsocket *wsi,
		enum libwebsocket_callback_reasons reason, void *user, void *in, size_t len)
{
	switch (reason) {
	case LWS_CALLBACK_CLIENT_RECEIVE:
		switch (clientState) {
		case CS_AUTH_INIT: {
			unsigned char challenge[32];

			if (len != 64) {
				ws_log("bad challenge length %u", (unsigned) len);
				exit(ERR_PROTOCOL_VIOLATION);
			}
			if (!hexDecode(in, challenge, sizeof(challenge))) {
				ws_log("bad challenge format");
				exit(ERR_PROTOCOL_VIOLATION);
			}

			if (!HMAC(EVP_sha256(), devKey, devKeyLen, challenge, sizeof(challenge), response, NULL)) {
				ws_log("error computing response");
				exit(ERR_INTERNAL);
			}

			clientState = CS_AUTH_COMPUTED;
			libwebsocket_callback_on_writable(context, wsi);
			break;
		}

		case CS_AUTH_SENT:
			if (len != sizeof(authResponseOK) - 1 || memcmp(in, authResponseOK, sizeof(authResponseOK) - 1)) {
				ws_log("authentication failed: got %*s", (int) len, in);
				exit(ERR_AUTHENTICATION_FAILED);
			}
			if (forkWhenReady)
				go_to_background();
			clientState = CS_BRIDGING;
			pollSet[0].events = POLLIN;
			break;

		case CS_BRIDGING:
			if (write(STDOUT_FILENO, in, len) < 0) {
				if (errno == ESPIPE) {
					exit(ERR_PIPE);
				} else {
					ws_log("write(1): %s", strerror(errno));
					exit(ERR_INTERNAL);
				}
			} else {
				char nl = '\n';
				if (write(STDOUT_FILENO, &nl, 1) < 0) {
					ws_log("write(1): %s", strerror(errno));
					exit(ERR_PIPE);
				}
			}
			break;
		}
		break;

	case LWS_CALLBACK_CLIENT_WRITEABLE:
		switch (clientState) {
		case CS_AUTH_COMPUTED: {
			unsigned char responseText[2 * sizeof(response) + LWS_SEND_BUFFER_PRE_PADDING + LWS_SEND_BUFFER_POST_PADDING];
			unsigned l = sizeof(response);

			hexEncode(response, (char*) responseText + LWS_SEND_BUFFER_PRE_PADDING, l);
			if (libwebsocket_write(wsi, responseText + LWS_SEND_BUFFER_PRE_PADDING, 2 * l, LWS_WRITE_TEXT) != 2 * l) {
				ws_log("error sending response");
				exit(ERR_NETWORK);
			}
			clientState = CS_AUTH_SENT;
			break;
		}

		case CS_BRIDGING: {
			unsigned char buffer[LWS_SEND_BUFFER_PRE_PADDING + LWS_SEND_BUFFER_POST_PADDING + sizeof(inputBuffer)];

			memcpy(buffer + LWS_SEND_BUFFER_PRE_PADDING, inputBuffer, inputBufferLength);
			int sent = libwebsocket_write(wsi, buffer + LWS_SEND_BUFFER_PRE_PADDING, inputBufferLength, LWS_WRITE_TEXT);
			if (sent != inputBufferLength) {
				ws_log("short send");
				exit(ERR_NETWORK);
			}
			inputBufferLength = 0;
			pollSet[0].events |= POLLIN;
		}
		}
		break;

	case LWS_CALLBACK_CLOSED:
		switch (clientState) {
		case CS_AUTH_SENT:
			ws_log("authentication failed");
			exit(ERR_AUTHENTICATION_FAILED);

		default:
			ws_log("peer closed connection");
			exit(ERR_NONE);
		}
		break;

	case LWS_CALLBACK_ADD_POLL_FD:
	case LWS_CALLBACK_DEL_POLL_FD:
		// poll descriptor is already reserved and will never be freed
		break;

	case LWS_CALLBACK_CHANGE_MODE_POLL_FD:
		pollSet[1].fd = ((struct libwebsocket_pollargs*) in)->fd;
		pollSet[1].events = ((struct libwebsocket_pollargs*) in)->events;
		pollSet[1].revents = ((struct libwebsocket_pollargs*) in)->prev_events;
		break;

	case LWS_CALLBACK_WSI_DESTROY:
		ws_log("websocket closed unexpectedly");
		exit(ERR_NETWORK);

	default:
		break;
	}

	return 0;
}

static void runDevice()
{
	struct libwebsocket_protocols protocols[] = {
		{
			.name = "msg/1/device",
			.callback = msg_1_device_callback,
			.rx_buffer_size = 4096,
		},
		{ 0 },
	};

	struct lws_context_creation_info ccinfo = {
		.port = CONTEXT_PORT_NO_LISTEN,
		.protocols = protocols,
		.uid = -1,
		.gid = -1,
		.ssl_ca_filepath = caPath,
	};

	lws_set_log_level(LLL_ERR | LLL_WARN, NULL);

	struct libwebsocket_context* context = libwebsocket_create_context(&ccinfo);
	if (!context) {
		ws_log("creating websocket context failed");
		exit(ERR_INTERNAL);
	}

	int useSSL = caPath ? 1 : 0;
	struct libwebsocket* socket = libwebsocket_client_connect(context, host, port, useSSL, wsPath,
			host, NULL, protocols[0].name, -1);
	if (!socket) {
		ws_log("connect failed");
		goto fail_context;
	}

	memset(pollSet, 0, sizeof(pollSet));
	pollSet[0].fd = 0;
	pollSet[0].events = 0;

	fcntl(0, F_SETFL, O_NONBLOCK);

	// run once to register the socket in pollSet
	libwebsocket_service(context, 0);

	while (poll(pollSet, 2, -1) >= 0 || errno == EINTR) {
		if (pollSet[0].revents & ~POLLIN)
			exit(ERR_PIPE);
		if (pollSet[0].revents & POLLIN) {
			char buffer;
			while (read(STDIN_FILENO, &buffer, 1) != -1) {
				if (buffer == '\r' || buffer == '\n') {
					break;
				}
				if (inputBufferLength >= sizeof(inputBuffer)) {
					ws_log("input overflow");
					exit(ERR_PROTOCOL_VIOLATION);
				}
				inputBuffer[inputBufferLength++] = buffer;
			}
			if (errno != EAGAIN && errno != EINPROGRESS && errno != 0) {
				perror("read(0)");
				exit(1);
			}
			if ((buffer == '\r' || buffer == '\n') && inputBufferLength > 0)
				libwebsocket_callback_on_writable(context, socket);
			pollSet[0].events = 0;
			pollSet[0].revents = 0;
		}
		libwebsocket_service(context, 0);
	}

	libwebsocket_context_destroy(context);
	return;

fail_context:
	libwebsocket_context_destroy(context);
	exit(ERR_NETWORK);
}

enum {
	WS_ARG_UNKNOWN = '?',

	WS_ARG_HOST,
	WS_ARG_PORT,
	WS_ARG_USER,
	WS_ARG_DEVICE,
	WS_ARG_KEY,
	WS_ARG_CA_PATH,
	WS_ARG_FORK,
	WS_ARG_HELP,
};

static const struct option options[] = {
	{ "host",   required_argument, NULL, WS_ARG_HOST },
	{ "port",   required_argument, NULL, WS_ARG_PORT },
	{ "user",   required_argument, NULL, WS_ARG_USER },
	{ "device", required_argument, NULL, WS_ARG_DEVICE },
	{ "key",    required_argument, NULL, WS_ARG_KEY },
	{ "capath", required_argument, NULL, WS_ARG_CA_PATH },
	{ "fork",   no_argument,       NULL, WS_ARG_FORK },
	{ "help",   no_argument,       NULL, WS_ARG_HELP },
	{ 0, },
};

static void usage(FILE* target)
{
	printf("Usage: ws --url <ws-url> --user <user> --device <device> --key <device-key> [in-fifo] [out-fifo]\n");
}

int main(int argc, char* argv[])
{
	int n = 0;
	unsigned argsFound = 0;

	if (argc < 2) {
		usage(stderr);
		return ERR_ARG;
	}

	while ((n = getopt_long(argc, argv, "", options, NULL)) != -1) {
		switch (n) {
		case WS_ARG_UNKNOWN:
			usage(stderr);
			return ERR_ARG;
		case WS_ARG_HELP:
			usage(stdout);
			return ERR_NONE;
		case WS_ARG_HOST:
			host = optarg;
			break;
		case WS_ARG_PORT: {
			char* ep;
			errno = 0;
			port = strtoul(optarg, &ep, 10);
			if (errno || port > 0xffff) {
				ws_log("port out of range");
				return ERR_ARG;
			}
			break;
		}
		case WS_ARG_USER:
			user = optarg;
			break;
		case WS_ARG_DEVICE:
			device = optarg;
			break;
		case WS_ARG_KEY: {
			size_t len = strlen(optarg + 2);
			if (len > sizeof(devKey)) {
				ws_log("--key argument invalid");
				return ERR_ARG;
			}
			if (optarg[0] == '0' && optarg[1] == 'x') {
				if (len % 2 || !hexDecode(optarg + 2, devKey, len / 2)) {
					ws_log("--key argument invalid");
					return ERR_ARG;
				}
				devKeyLen = len / 2;
			} else {
				strcpy((char*) devKey, optarg);
				devKeyLen = len;
			}
			break;
		}
		case WS_ARG_CA_PATH:
			caPath = optarg;
			break;
		case WS_ARG_FORK:
			forkWhenReady = true;
			break;
		}

		argsFound |= 1 << (n - WS_ARG_UNKNOWN);
	}

	if (optind != argc) {
		if (optind != argc - 2) {
			usage(stderr);
			return ERR_ARG;
		}

		int fd = open(argv[optind], O_RDONLY);
		if (fd < 0 || dup2(fd, STDIN_FILENO) < 0 || close(fd)) {
			ws_log("could not open in fifo; %s", strerror(errno));
			return ERR_ARG;
		}

		fd = open(argv[optind + 1], O_WRONLY);
		if (fd < 0 || dup2(fd, STDOUT_FILENO) < 0 || close(fd)) {
			ws_log("could not open out fifo; %s", strerror(errno));
			return ERR_ARG;
		}
	}

#define REQUIRE(arg, name) \
	do { \
		if (!(argsFound & (1 << (WS_ARG_ ## arg - WS_ARG_UNKNOWN)))) { \
			ws_log(name " missing\n"); \
			return 1; \
		} \
	} while (0)

	REQUIRE(HOST, "--host");
	REQUIRE(PORT, "--port");
	REQUIRE(USER, "--user");
	REQUIRE(DEVICE, "--device");
	REQUIRE(KEY, "--key");

#undef REQUIRE

	if (snprintf(wsPath, sizeof(wsPath), "/ws/device/%s/%s", user, device) >= sizeof(wsPath)) {
		ws_log("ws path way too long, check args");
		return ERR_ARG;
	}

	runDevice();

	return ERR_NONE;
}
