create table if not exists mercenary(
    id int not null auto_increment,
    unit_id varchar(255) not null UNIQUE,
    name varchar(255) not null,
    mythium_cost int,
    icon_path varchar(255) not null,
    version varchar(16) not null,
    primary key(id)
);