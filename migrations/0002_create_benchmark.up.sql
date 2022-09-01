create table if not exists benchmark(
    id int not null auto_increment,
    wave int not null,
    unit_id varchar(255) not null,
    value int not null,
    primary key(id)
);