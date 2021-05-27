# grpcSandbox
Learning gRPC

In order to use, set enviromental variables
```bash
export GRPC_SANDBOX_ROLE=role
export GRPC_SANDBOX_PASSWORD=pass
export GRPC_SANDBOX_DB_NAME=database
```
...and create the following tables
```bash
create table users
(
    id         uuid default gen_random_uuid() primary key,
    name       text      not null,
    age        smallint  not null,
    type       smallint  not null,
    created_at timestamp not null,
    updated_at timestamp not null
);
create table items
(
    id         uuid default gen_random_uuid() primary key,
    name       text      not null,
    user_id    uuid      not null references users (id) on delete cascade,
    created_at timestamp not null,
    updated_at timestamp not null
);
```
