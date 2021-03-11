CREATE DATABASE grafana DEFAULT CHARACTER SET utf8mb4;
CREATE USER 'chainmaker'@'%' IDENTIFIED BY 'chainmaker';
GRANT all privileges ON grafana.* TO 'chainmaker'@'%';
FLUSH PRIVILEGES;