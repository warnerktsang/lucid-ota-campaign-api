CREATE TABLE campaign
(
    id             VARCHAR PRIMARY KEY,
    name           VARCHAR NOT NULL,
    target_version VARCHAR(32) NOT NULL,
    fleet_id       VARCHAR(36) NOT NULL,
    created_at     TIMESTAMP NOT NULL,
    updated_at     TIMESTAMP NOT NULL
);
