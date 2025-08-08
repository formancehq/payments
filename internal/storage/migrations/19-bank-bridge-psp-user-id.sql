alter table psu_bank_bridges
    add column psp_user_id text;

create index idx_psu_bank_bridges_psp_user_id on psu_bank_bridges (psp_user_id);