--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.1
-- Dumped by pg_dump version 9.5.1

-- Started on 2016-03-23 14:30:36 CET

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 1 (class 3079 OID 12395)
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- TOC entry 2268 (class 0 OID 0)
-- Dependencies: 1
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

--
-- TOC entry 199 (class 1255 OID 18801)
-- Name: do_aggregate(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION do_aggregate() RETURNS bigint
    LANGUAGE sql
    AS $$
with updates as (
	delete from measure_raw
	returning
		sensor,
		"timestamp",
		value

), do_update_s as (
	insert into measure_aggregated_seconds as m
	select
		date_trunc('second', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		1
	from updates
	group by
		ts,
		sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
), do_update_m as (
	insert into measure_aggregated_minutes as m
	select
		date_trunc('minute', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		2
	from updates
	group by ts, sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
), do_update_h as (
	insert into measure_aggregated_hours as m
	select
		date_trunc('hour', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		3
	from updates
	group by ts, sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
), do_update_d as (
	insert into measure_aggregated_days as m
	select
		date_trunc('day', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		4
	from updates
	group by ts, sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
), do_update_w as (
	insert into measure_aggregated_weeks as m
	select
		date_trunc('week', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		5
	from updates
	group by ts, sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
), do_update_mo as (
	insert into measure_aggregated_months as m
	select
		date_trunc('months', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		6
	from updates
	group by ts, sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
), do_update_y as (
	insert into measure_aggregated_years as m
	select
		date_trunc('year', "timestamp") as ts,
		sum(value) as sum,
		count(value) as count,
		sensor,
		7
	from updates
	group by ts, sensor
	on conflict ("timestamp", sensor) do update
	set sum = m.sum + excluded.sum, count = m.count + excluded.count
	returning 1
)

select count(*) from do_update_y;$$;


--
-- TOC entry 212 (class 1255 OID 16402)
-- Name: do_remove_old_values(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION do_remove_old_values() RETURNS void
    LANGUAGE plpgsql
    AS $$
BEGIN
DELETE FROM measure_aggregated_seconds
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[1]);
DELETE FROM measure_aggregated_minutes
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[2]);
DELETE FROM measure_aggregated_hours
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[3]);
DELETE FROM measure_aggregated_days
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[4]);
DELETE FROM measure_aggregated_weeks
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[5]);
DELETE FROM measure_aggregated_months
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[6]);
DELETE FROM measure_aggregated_years
USING users u, sensors s
WHERE sensor = s.sensor_seq
	AND s.user_id = u.user_id
	AND "timestamp" < (now() - u.remove_data_after[7]);
END;$$;


SET default_tablespace = '';

SET default_with_oids = false;

--
-- TOC entry 181 (class 1259 OID 16544)
-- Name: devices; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE devices (
    device_id character varying NOT NULL,
    key bytea NOT NULL,
    name character varying NOT NULL,
    user_id character varying NOT NULL,
    is_virtual boolean DEFAULT false NOT NULL
);


--
-- TOC entry 182 (class 1259 OID 16550)
-- Name: groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE groups (
    group_id character varying NOT NULL
);


--
-- TOC entry 183 (class 1259 OID 16556)
-- Name: measure_aggregated_days; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_days (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 184 (class 1259 OID 16559)
-- Name: measure_aggregated_hours; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_hours (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 185 (class 1259 OID 16562)
-- Name: measure_aggregated_minutes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_minutes (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 186 (class 1259 OID 16565)
-- Name: measure_aggregated_months; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_months (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 187 (class 1259 OID 16568)
-- Name: measure_aggregated_seconds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_seconds (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 188 (class 1259 OID 16571)
-- Name: measure_aggregated_weeks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_weeks (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 189 (class 1259 OID 16574)
-- Name: measure_aggregated_years; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE measure_aggregated_years (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


--
-- TOC entry 190 (class 1259 OID 16577)
-- Name: measure_raw; Type: TABLE; Schema: public; Owner: -
--

CREATE UNLOGGED TABLE measure_raw (
    "timestamp" timestamp with time zone,
    value double precision,
    sensor bigint
);


--
-- TOC entry 191 (class 1259 OID 16580)
-- Name: sensor_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE sensor_groups (
    sensor_seq bigint NOT NULL,
    group_id character varying NOT NULL
);


--
-- TOC entry 192 (class 1259 OID 16586)
-- Name: sensors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE sensors (
    sensor_id character varying NOT NULL,
    device_id character varying NOT NULL,
    user_id character varying NOT NULL,
    name character varying NOT NULL,
    port integer NOT NULL,
    unit character varying NOT NULL,
    sensor_seq bigint NOT NULL,
    is_virtual boolean DEFAULT false NOT NULL,
    factor double precision DEFAULT 1.0 NOT NULL
);


--
-- TOC entry 193 (class 1259 OID 16593)
-- Name: sensors_sensor_seq_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE sensors_sensor_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2269 (class 0 OID 0)
-- Dependencies: 193
-- Name: sensors_sensor_seq_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE sensors_sensor_seq_seq OWNED BY sensors.sensor_seq;


--
-- TOC entry 194 (class 1259 OID 16595)
-- Name: user_groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE user_groups (
    user_id character varying NOT NULL,
    group_id character varying NOT NULL,
    is_admin boolean DEFAULT false
);


--
-- TOC entry 195 (class 1259 OID 16602)
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE users (
    user_id character varying NOT NULL,
    pw_hash bytea,
    is_admin boolean NOT NULL,
    remove_data_after interval[]
);


--
-- TOC entry 198 (class 1259 OID 16723)
-- Name: virtual_sensor_sensors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE virtual_sensor_sensors (
    sensor_seq bigint NOT NULL,
    vsensor_id bigint NOT NULL,
    symbol smallint NOT NULL
);


--
-- TOC entry 197 (class 1259 OID 16695)
-- Name: virtual_sensors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE virtual_sensors (
    vsensor_id bigint NOT NULL,
    formula character varying,
    representing_sensor bigint NOT NULL
);


--
-- TOC entry 196 (class 1259 OID 16693)
-- Name: virtual_sensors_vsensor_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE virtual_sensors_vsensor_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 2270 (class 0 OID 0)
-- Dependencies: 196
-- Name: virtual_sensors_vsensor_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE virtual_sensors_vsensor_id_seq OWNED BY virtual_sensors.vsensor_id;


--
-- TOC entry 2092 (class 2604 OID 16406)
-- Name: sensor_seq; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensors ALTER COLUMN sensor_seq SET DEFAULT nextval('sensors_sensor_seq_seq'::regclass);


--
-- TOC entry 2095 (class 2604 OID 16407)
-- Name: vsensor_id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensors ALTER COLUMN vsensor_id SET DEFAULT nextval('virtual_sensors_vsensor_id_seq'::regclass);


--
-- TOC entry 2101 (class 2606 OID 18094)
-- Name: days_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_days
    ADD CONSTRAINT days_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2097 (class 2606 OID 16408)
-- Name: device_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY devices
    ADD CONSTRAINT device_pk PRIMARY KEY (device_id, user_id);


--
-- TOC entry 2099 (class 2606 OID 16409)
-- Name: groups_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pk PRIMARY KEY (group_id);


--
-- TOC entry 2103 (class 2606 OID 18096)
-- Name: hours_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_hours
    ADD CONSTRAINT hours_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2105 (class 2606 OID 18098)
-- Name: minutes_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_minutes
    ADD CONSTRAINT minutes_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2107 (class 2606 OID 18100)
-- Name: months_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_months
    ADD CONSTRAINT months_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2109 (class 2606 OID 18102)
-- Name: seconds_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_seconds
    ADD CONSTRAINT seconds_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2126 (class 2606 OID 16410)
-- Name: sensor_1to1; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensors
    ADD CONSTRAINT sensor_1to1 UNIQUE (representing_sensor);


--
-- TOC entry 2116 (class 2606 OID 16411)
-- Name: sensor_groups_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensor_groups
    ADD CONSTRAINT sensor_groups_pk PRIMARY KEY (sensor_seq, group_id);


--
-- TOC entry 2118 (class 2606 OID 16412)
-- Name: sensor_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT sensor_pk PRIMARY KEY (sensor_id, device_id, user_id);


--
-- TOC entry 2120 (class 2606 OID 16413)
-- Name: seq_unique; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT seq_unique UNIQUE (sensor_seq);


--
-- TOC entry 2122 (class 2606 OID 16414)
-- Name: user_group_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT user_group_pk PRIMARY KEY (group_id, user_id);


--
-- TOC entry 2124 (class 2606 OID 16415)
-- Name: users_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pk PRIMARY KEY (user_id);


--
-- TOC entry 2130 (class 2606 OID 19209)
-- Name: virtual_sensor_sensors_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensor_sensors
    ADD CONSTRAINT virtual_sensor_sensors_pk PRIMARY KEY (sensor_seq, vsensor_id, symbol);


--
-- TOC entry 2128 (class 2606 OID 16417)
-- Name: virtual_sensors_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensors
    ADD CONSTRAINT virtual_sensors_pk PRIMARY KEY (vsensor_id);


--
-- TOC entry 2111 (class 2606 OID 18104)
-- Name: weeks_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_weeks
    ADD CONSTRAINT weeks_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2113 (class 2606 OID 18106)
-- Name: years_pk; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_years
    ADD CONSTRAINT years_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 2114 (class 1259 OID 17593)
-- Name: brin_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX brin_index ON measure_raw USING brin ("timestamp");


--
-- TOC entry 2142 (class 2606 OID 16418)
-- Name: device_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT device_fk FOREIGN KEY (device_id, user_id) REFERENCES devices(device_id, user_id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 2140 (class 2606 OID 16423)
-- Name: group_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensor_groups
    ADD CONSTRAINT group_fk FOREIGN KEY (group_id) REFERENCES groups(group_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 2143 (class 2606 OID 16428)
-- Name: groups_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT groups_fk FOREIGN KEY (group_id) REFERENCES groups(group_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 2141 (class 2606 OID 16473)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY sensor_groups
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor_seq) REFERENCES sensors(sensor_seq) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 2145 (class 2606 OID 16478)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensors
    ADD CONSTRAINT sensor_fk FOREIGN KEY (representing_sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 2136 (class 2606 OID 17687)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_seconds
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2134 (class 2606 OID 17736)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_minutes
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2133 (class 2606 OID 17741)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_hours
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2132 (class 2606 OID 17746)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_days
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2137 (class 2606 OID 17751)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_weeks
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2135 (class 2606 OID 17756)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_months
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2138 (class 2606 OID 17761)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_aggregated_years
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2139 (class 2606 OID 17794)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY measure_raw
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;


--
-- TOC entry 2146 (class 2606 OID 16483)
-- Name: sensors_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensor_sensors
    ADD CONSTRAINT sensors_fk FOREIGN KEY (sensor_seq) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 2131 (class 2606 OID 16488)
-- Name: user_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY devices
    ADD CONSTRAINT user_fk FOREIGN KEY (user_id) REFERENCES users(user_id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 2144 (class 2606 OID 16493)
-- Name: users_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT users_fk FOREIGN KEY (user_id) REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 2147 (class 2606 OID 16498)
-- Name: virtual_sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY virtual_sensor_sensors
    ADD CONSTRAINT virtual_sensor_fk FOREIGN KEY (vsensor_id) REFERENCES virtual_sensors(vsensor_id) ON UPDATE RESTRICT ON DELETE CASCADE;


-- Completed on 2016-03-23 14:30:36 CET

--
-- PostgreSQL database dump complete
--

