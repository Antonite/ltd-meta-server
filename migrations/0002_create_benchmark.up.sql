create table if not exists benchmark(
    id int not null auto_increment,
    wave int not null,
    unit_id int not null,
    value int not null,
    primary key(id),
    foreign key(unit_id) references unit(id)
);