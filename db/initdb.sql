--
-- PostgreSQL database dump
--

-- Dumped from database version 9.3.9
-- Dumped by pg_dump version 9.3.9
-- Started on 2015-07-29 15:07:03 CEST

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

--
-- TOC entry 176 (class 3079 OID 11791)
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: 
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- TOC entry 2014 (class 0 OID 0)
-- Dependencies: 176
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- TOC entry 173 (class 1259 OID 16633)
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
-- TOC entry 171 (class 1259 OID 16406)
-- Name: measure_avg_1m; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE measure_avg_1m (
    "SensorID" character varying NOT NULL,
    "Timestamp" timestamp with time zone NOT NULL,
    "Sum" double precision,
    "Count" bigint
);


ALTER TABLE public.measure_avg_1m OWNER TO msgp;

--
-- TOC entry 170 (class 1259 OID 16386)
-- Name: measure_raw; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE measure_raw (
    "timestamp" timestamp with time zone NOT NULL,
    value double precision NOT NULL,
    sensor bigint NOT NULL
);


ALTER TABLE public.measure_raw OWNER TO msgp;

--
-- TOC entry 174 (class 1259 OID 16646)
-- Name: sensors; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE sensors (
    sensor_id character varying NOT NULL,
    device_id character varying NOT NULL,
    user_id character varying NOT NULL,
    name character varying NOT NULL,
    port integer NOT NULL,
    unit character varying NOT NULL,
    sensor_seq bigint NOT NULL
);


ALTER TABLE public.sensors OWNER TO msgp;

--
-- TOC entry 175 (class 1259 OID 16659)
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
-- TOC entry 2015 (class 0 OID 0)
-- Dependencies: 175
-- Name: sensors_sensor_seq_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: msgp
--

ALTER SEQUENCE sensors_sensor_seq_seq OWNED BY sensors.sensor_seq;


--
-- TOC entry 172 (class 1259 OID 16625)
-- Name: users; Type: TABLE; Schema: public; Owner: msgp; Tablespace: 
--

CREATE TABLE users (
    user_id character varying NOT NULL,
    pw_hash bytea,
    is_admin boolean NOT NULL
);


ALTER TABLE public.users OWNER TO msgp;

--
-- TOC entry 1884 (class 2604 OID 16661)
-- Name: sensor_seq; Type: DEFAULT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY sensors ALTER COLUMN sensor_seq SET DEFAULT nextval('sensors_sensor_seq_seq'::regclass);


--
-- TOC entry 1888 (class 2606 OID 16413)
-- Name: ValueID_avg_1m; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY measure_avg_1m
    ADD CONSTRAINT "ValueID_avg_1m" PRIMARY KEY ("SensorID", "Timestamp");


--
-- TOC entry 1892 (class 2606 OID 16640)
-- Name: device_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY devices
    ADD CONSTRAINT device_pk PRIMARY KEY (device_id, user_id);


--
-- TOC entry 1886 (class 2606 OID 16676)
-- Name: measrue_raw_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY measure_raw
    ADD CONSTRAINT measrue_raw_pk PRIMARY KEY ("timestamp", sensor);


--
-- TOC entry 1894 (class 2606 OID 16685)
-- Name: sensor_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT sensor_pk PRIMARY KEY (sensor_id, device_id, user_id);


--
-- TOC entry 1896 (class 2606 OID 16670)
-- Name: seq_unique; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT seq_unique UNIQUE (sensor_seq);


--
-- TOC entry 1890 (class 2606 OID 16632)
-- Name: users_pk; Type: CONSTRAINT; Schema: public; Owner: msgp; Tablespace: 
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pk PRIMARY KEY (user_id);


--
-- TOC entry 1899 (class 2606 OID 16654)
-- Name: device_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY sensors
    ADD CONSTRAINT device_fk FOREIGN KEY (device_id, user_id) REFERENCES devices(device_id, user_id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1897 (class 2606 OID 16677)
-- Name: sensor_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY measure_raw
    ADD CONSTRAINT sensor_fk FOREIGN KEY (sensor) REFERENCES sensors(sensor_seq) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 1898 (class 2606 OID 16641)
-- Name: user_fk; Type: FK CONSTRAINT; Schema: public; Owner: msgp
--

ALTER TABLE ONLY devices
    ADD CONSTRAINT user_fk FOREIGN KEY (user_id) REFERENCES users(user_id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- TOC entry 2013 (class 0 OID 0)
-- Dependencies: 5
-- Name: public; Type: ACL; Schema: -; Owner: postgres
--

REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM postgres;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2015-07-29 15:07:03 CEST

--
-- PostgreSQL database dump complete
--

