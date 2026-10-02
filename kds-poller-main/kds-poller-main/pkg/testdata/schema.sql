CREATE TABLE `kds_files` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT
,  `hash` varchar(255) NOT NULL
,  `created` datetime NOT NULL DEFAULT current_timestamp
,  `filename` varchar(100) NOT NULL
);
CREATE TABLE `kds_items` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT
,  `qty` integer NOT NULL
,  `name` varchar(255) NOT NULL
,  `status` varchar(50) DEFAULT 'OPEN'
,  `kds_screen_id` integer NOT NULL
,  `kds_order_id` integer NOT NULL
,  `created` timestamp NULL DEFAULT current_timestamp
,  `iskiddies` integer NOT NULL DEFAULT 0
,  `prep_mins` integer DEFAULT NULL
,  `prep_secs` integer DEFAULT NULL
,  `suborder` integer DEFAULT 0
, qty_raw REAL, in_production BOOLEAN DEFAULT 0, production_started_at DATETIME, production_ended_at DATETIME, pushed_to_ods BOOLEAN DEFAULT 0, pushed_to_ods_at DATETIME, sent_to_table BOOLEAN DEFAULT 0, sent_to_table_at DATETIME);
CREATE TABLE `kds_orders` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT
,  `table` varchar(255) NOT NULL
,  `waiter` varchar(255) NOT NULL
,  `status` varchar(50) NOT NULL DEFAULT '''OPEN'''
,  `profile` varchar(50) DEFAULT NULL
,  `order_forday` integer DEFAULT NULL
,  `created` timestamp NULL DEFAULT current_timestamp
,  `reference` varchar(100) DEFAULT NULL
,  `completed` timestamp NULL DEFAULT NULL
,  `suborders` integer DEFAULT 0
,  `pass_status` varchar(50) DEFAULT 'OPEN'
,  `consumer` varchar(100) DEFAULT NULL
,  `delivery` varchar(100) DEFAULT NULL
, transfer_note TEXT);
CREATE TABLE `kds_screens` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT
,  `name` varchar(255) NOT NULL
,  `port` varchar(255) NOT NULL
, assigned_pass_terminal TEXT, is_printer_only BOOLEAN DEFAULT 0);
CREATE TABLE `kds_subitems` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT
,  `qty` integer NOT NULL
,  `name` varchar(255) NOT NULL
,  `status` varchar(50) DEFAULT NULL
,  `kds_item_id` integer NOT NULL
, qty_raw REAL);
CREATE TABLE kds_pass_terminals (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, last_seen DATETIME);
