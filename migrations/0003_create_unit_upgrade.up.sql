create table if not exists unit_upgrade(
    id int not null auto_increment,
    unit_id int not null,
    upgrade_id int not null,
    primary key(id),
    CONSTRAINT fk_unit_upgrade_unit_id foreign key(unit_id) references unit(id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT fk_unit_upgrade_upgrades_id foreign key(upgrade_id) references unit(id) ON UPDATE CASCADE ON DELETE CASCADE
);