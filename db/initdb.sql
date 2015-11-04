--
-- PostgreSQL database dump
--

-- Dumped from database version 9.3.9
-- Dumped by pg_dump version 9.3.9
-- Started on 2015-11-03 14:56:47 CET

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- TOC entry 185 (class 3079 OID 11791)
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- TOC entry 2069 (class 0 OID 0)
-- Dependencies: 185
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

--
-- TOC entry 199 (class 1255 OID 17846)
-- Name: do_aggregate(); Type: FUNCTION; Schema: public; Owner: msgp
--

CREATE FUNCTION do_aggregate() RETURNS bigint
    LANGUAGE sql
    AS $$with timestamps_s as (
	select distinct
		sensor,
		date_trunc('second', "timestamp") as ts
	from measure_raw
), non_exist_s as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_seconds m
	right join timestamps_s t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_s as (
	insert into measure_aggregated_seconds 
	select 
		ts, 0, 0,sensor, 0
	from non_exist_s
	returning
		"timestamp", sensor
), timestamps_m as (
	select distinct
		sensor,
		date_trunc('minute', "timestamp") as ts
	from insert_s
), non_exist_m as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_minutes m
	right join timestamps_m t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_m as (
	insert into measure_aggregated_minutes 
	select 
		ts, 0, 0,sensor, 1
	from non_exist_m
	returning
		"timestamp", sensor
), timestamps_h as (
	select distinct
		sensor,
		date_trunc('hour', "timestamp") as ts
	from insert_m
), non_exist_h as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_hours m
	right join timestamps_h t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_h as (
	insert into measure_aggregated_hours 
	select 
		ts, 0, 0,sensor, 2
	from non_exist_h
	returning
		"timestamp", sensor
), timestamps_d as (
	select distinct
		sensor,
		date_trunc('day', "timestamp") as ts
	from insert_h
), non_exist_d as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_days m
	right join timestamps_d t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_d as (
	insert into measure_aggregated_days
	select 
		ts, 0, 0,sensor, 3
	from non_exist_d
	returning 
		"timestamp", sensor
),timestamps_w as (
	select distinct
		sensor,
		date_trunc('week', "timestamp") as ts
	from insert_d
), non_exist_w as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_weeks m
	right join timestamps_w t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_w as (
	insert into measure_aggregated_weeks
	select 
		ts, 0, 0,sensor, 4
	from non_exist_w
	returning 
		"timestamp", sensor
), timestamps_mo as (
	select distinct
		sensor,
		date_trunc('month', "timestamp") as ts
	from insert_w
), non_exist_mo as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_months m
	right join timestamps_mo t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_mo as (
	insert into measure_aggregated_months
	select 
		ts, 0, 0,sensor, 5
	from non_exist_mo
	returning
		"timestamp", sensor
), timestamps_y as (
	select distinct
		sensor,
		date_trunc('year', "timestamp") as ts
	from insert_mo
), non_exist_y as (
	select
		t.sensor,
		t.ts
	from measure_aggregated_years m
	right join timestamps_y t on (
		m.sensor = t.sensor and
		m."timestamp" = t.ts
	)
	where m.sensor is null
), insert_y as (
	insert into measure_aggregated_years
	select 
		ts, 0, 0,sensor, 6
	from non_exist_y
	returning 1
)
select count(*) from insert_y;

with updates as (
	delete from measure_raw
	returning
		sensor,
		date_trunc('second', "timestamp") as "timestamp",
		value
), grouped_s as (
	select
		"timestamp",
		count(value),
		sum(value),
		sensor
	from updates
	group by
		"timestamp",
		sensor
), do_update_s as (
	update measure_aggregated_seconds m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_s g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning
		date_trunc('minute', g."timestamp") as "timestamp", g.sum, g.count, g.sensor
), grouped_m as (
	select
		"timestamp",
		sum(count) as count,
		sum(sum) as sum,
		sensor,
		1
	from do_update_s
	group by
		"timestamp",
		sensor
), do_update_m as (
	update measure_aggregated_minutes m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_m g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning
		date_trunc('hour', g."timestamp") as "timestamp", g.sum, g.count, g.sensor
), grouped_h as (
	select
		"timestamp",
		sum(count) as count,
		sum(sum) as sum,
		sensor
	from do_update_m
	group by
		"timestamp",
		sensor
), do_update_h as (
	update measure_aggregated_hours m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_h g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning
		date_trunc('day', g."timestamp") as "timestamp", g.sum, g.count, g.sensor
),grouped_d as (
	select
		"timestamp",
		sum(count) as count,
		sum(sum) as sum,
		sensor
	from do_update_h
	group by
		"timestamp",
		sensor
), do_update_d as (
	update measure_aggregated_days m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_d g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning
		date_trunc('week', g."timestamp") as "timestamp", g.sum, g.count, g.sensor
), grouped_w as (
	select
		"timestamp",
		sum(count) as count,
		sum(sum) as sum,
		sensor
	from do_update_d
	group by
		"timestamp",
		sensor
), do_update_w as (
	update measure_aggregated_weeks m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_w g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning
		date_trunc('month', g."timestamp") as "timestamp", g.sum, g.count, g.sensor
), grouped_mo as (
	select
		"timestamp",
		sum(count) as count,
		sum(sum) as sum,
		sensor
	from do_update_w
	group by
		"timestamp",
		sensor
), do_update_mo as (
	update measure_aggregated_months m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_mo g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning
		date_trunc('year', g."timestamp") as "timestamp", g.sum, g.count, g.sensor
),grouped_y as (
	select
		"timestamp",
		sum(count) as count,
		sum(sum) as sum,
		sensor
	from do_update_mo
	group by
		"timestamp",
		sensor
), do_update_y as (
	update measure_aggregated_years m
	set sum = m.sum + g.sum, count = m.count + g.count
	from grouped_y g
	where m.sensor = g.sensor and m."timestamp" = g."timestamp"
	returning 1
)
select count(*) from do_update_y;$$;


ALTER FUNCTION public.do_aggregate() OWNER TO msgp;

--
-- TOC entry 200 (class 1255 OID 18086)
-- Name: do_remove_old_values(); Type: FUNCTION; Schema: public; Owner: msgp
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


ALTER FUNCTION public.do_remove_old_values() OWNER TO msgp;

--
-- TOC entry 198 (class 1255 OID 18591)
-- Name: measure_raw_insert_trigger(); Type: FUNCTION; Schema: public; Owner: msgp
--

CREATE FUNCTION measure_raw_insert_trigger() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NOT EXISTS(SELECT 1 FROM measure_updates WHERE sensor = NEW.sensor AND date_trunc('minute', NEW.timestamp) = timestamp) THEN
	INSERT INTO measure_updates VALUES (date_trunc('minute',NEW.timestamp), NEW.sensor, true);
    END IF;
    RETURN NULL;
END;
$$;


ALTER FUNCTION public.measure_raw_insert_trigger() OWNER TO msgp;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- TOC entry 171 (class 1259 OID 16633)
-- Name: devices; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE devices (
    device_id character varying NOT NULL,
    key bytea NOT NULL,
    name character varying NOT NULL,
    user_id character varying NOT NULL
);


ALTER TABLE public.devices OWNER TO msgp;

--
-- TOC entry 175 (class 1259 OID 18087)
-- Name: groups; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE groups (
    group_id character varying NOT NULL
);


ALTER TABLE public.groups OWNER TO msgp;

--
-- TOC entry 179 (class 1259 OID 18358)
-- Name: measure_aggregated_days; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_days (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_days OWNER TO msgp;

--
-- TOC entry 178 (class 1259 OID 18350)
-- Name: measure_aggregated_hours; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_hours (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_hours OWNER TO msgp;

--
-- TOC entry 177 (class 1259 OID 18342)
-- Name: measure_aggregated_minutes; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_minutes (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_minutes OWNER TO msgp;

--
-- TOC entry 181 (class 1259 OID 18375)
-- Name: measure_aggregated_months; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_months (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_months OWNER TO msgp;

--
-- TOC entry 176 (class 1259 OID 18334)
-- Name: measure_aggregated_seconds; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_seconds (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_seconds OWNER TO msgp;

--
-- TOC entry 180 (class 1259 OID 18366)
-- Name: measure_aggregated_weeks; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_weeks (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_weeks OWNER TO msgp;

--
-- TOC entry 182 (class 1259 OID 18383)
-- Name: measure_aggregated_years; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_aggregated_years (
    "timestamp" timestamp with time zone NOT NULL,
    sum double precision,
    count bigint,
    sensor bigint NOT NULL,
    "precision" integer
);


ALTER TABLE public.measure_aggregated_years OWNER TO msgp;

--
-- TOC entry 174 (class 1259 OID 18054)
-- Name: measure_raw; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE UNLOGGED TABLE measure_raw (
    "timestamp" timestamp with time zone,
    value double precision,
    sensor bigint
);


ALTER TABLE public.measure_raw OWNER TO msgp;

--
-- TOC entry 184 (class 1259 OID 18506)
-- Name: sensor_groups; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE sensor_groups (
    sensor_seq bigint NOT NULL,
    group_id character varying NOT NULL
);


ALTER TABLE public.sensor_groups OWNER TO msgp;

--
-- TOC entry 172 (class 1259 OID 16646)
-- Name: sensors; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE sensors (
    sensor_id character varying NOT NULL,
    device_id character varying NOT NULL,
    user_id character varying NOT NULL,
    name character varying NOT NULL,
    port integer NOT NULL,
    unit character varying NOT NULL,
    sensor_seq bigint NOT NULL,
    is_virtual boolean DEFAULT false NOT NULL
);


ALTER TABLE public.sensors OWNER TO msgp;

--
-- TOC entry 173 (class 1259 OID 16659)
-- Name: sensors_sensor_seq_seq; Type: SEQUENCE; Schema: public; Owner: msgp
--

CREATE SEQUENCE sensors_sensor_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.sensors_sensor_seq_seq OWNER TO msgp;

--
-- TOC entry 2070 (class 0 OID 0)
-- Dependencies: 173
-- Name: sensors_sensor_seq_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: msgp
--

ALTER SEQUENCE sensors_sensor_seq_seq OWNED BY sensors.sensor_seq;


--
-- TOC entry 183 (class 1259 OID 18433)
-- Name: user_groups; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE user_groups (
    user_id character varying NOT NULL,
    group_id character varying NOT NULL,
    is_admin boolean DEFAULT false
);


ALTER TABLE public.user_groups OWNER TO msgp;

--
-- TOC entry 170 (class 1259 OID 16625)
-- Name: users; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE users (
    user_id character varying NOT NULL,
    pw_hash bytea,
    is_admin boolean NOT NULL,
    remove_data_after interval[]
);


ALTER TABLE public.users OWNER TO msgp;

--
-- TOC entry 1924 (class 2604 OID 16661)
-- Name: sensor_seq; Type: DEFAULT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY sensors ALTER COLUMN sensor_seq SET DEFAULT nextval('sensors_sensor_seq_seq'::regclass);


--
-- TOC entry 1930 (class 2606 OID 16640)
-- Name: device_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY devices
    ADD CONSTRAINT device_pk PRIMARY KEY (device_id, user_id);


--
-- TOC entry 1936 (class 2606 OID 18094)
-- Name: groups_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY groups
    ADD CONSTRAINT groups_pk PRIMARY KEY (group_id);


--
-- TOC entry 1940 (class 2606 OID 18513)
-- Name: sensor_groups_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY sensor_groups
    ADD CONSTRAINT sensor_groups_pk PRIMARY KEY (sensor_seq, group_id);


--
-- TOC entry 1932 (class 2606 OID 16685)
-- Name: sensor_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT sensor_pk PRIMARY KEY (sensor_id, device_id, user_id);


--
-- TOC entry 1934 (class 2606 OID 16670)
-- Name: seq_unique; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT seq_unique UNIQUE (sensor_seq);


--
-- TOC entry 1938 (class 2606 OID 18440)
-- Name: user_group_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT user_group_pk PRIMARY KEY (group_id, user_id);


--
-- TOC entry 1928 (class 2606 OID 16632)
-- Name: users_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pk PRIMARY KEY (user_id);


--
-- TOC entry 1942 (class 2606 OID 16654)
-- Name: device_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT device_fk FOREIGN KEY (device_id, user_id) REFERENCES devices(device_id, user_id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1953 (class 2606 OID 18514)
-- Name: group_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY sensor_groups
    ADD CONSTRAINT group_fk FOREIGN KEY (group_id) REFERENCES groups(group_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 1952 (class 2606 OID 18456)
-- Name: groups_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT groups_fk FOREIGN KEY (group_id) REFERENCES groups(group_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 1943 (class 2606 OID 18057)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_raw
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1944 (class 2606 OID 18337)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_seconds
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1945 (class 2606 OID 18345)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_minutes
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1946 (class 2606 OID 18353)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_hours
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1947 (class 2606 OID 18361)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_days
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1948 (class 2606 OID 18369)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_weeks
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1949 (class 2606 OID 18378)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_months
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1950 (class 2606 OID 18386)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_aggregated_years
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1954 (class 2606 OID 18519)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY sensor_groups
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor_seq) REFERENCES sensors(sensor_seq) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 1941 (class 2606 OID 16641)
-- Name: user_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY devices
    ADD CONSTRAINT user_fk FOREIGN KEY (user_id) REFERENCES users(user_id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1951 (class 2606 OID 18451)
-- Name: users_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY user_groups
    ADD CONSTRAINT users_fk FOREIGN KEY (user_id) REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- TOC entry 2068 (class 0 OID 0)
-- Dependencies: 5
-- Name: public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM postgres;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2015-11-03 14:56:47 CET

--
-- PostgreSQL database dump complete
--

