#!/bin/bash
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="project-sem-1"
DB_USER="validator"
DB_PASSWORD="val1dat0r"    

go mod download
go mod tidy

sleep 5

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres <<EOF
DROP DATABASE IF EXISTS "$DB_NAME";
CREATE DATABASE "$DB_NAME";
CREATE TABLE prices (
    id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    create_date DATE NOT NULL
);
EOF
