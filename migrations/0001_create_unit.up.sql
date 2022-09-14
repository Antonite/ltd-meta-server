create table if not exists unit(
    id int not null auto_increment,
    unit_id varchar(255) not null UNIQUE,
    name varchar(255) not null,
    total_value int,
    usable boolean not null default 0,
    icon_path varchar(255) not null,
    version varchar(16) not null,
    primary key(id)
);