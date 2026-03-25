-- Откат миграции
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";