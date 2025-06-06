-- Luna IoT Server Database Setup Script
-- Execute this script in PostgreSQL to create the database and initial data

-- Create database (run this as postgres superuser)
-- CREATE DATABASE luna_iot;

-- Connect to luna_iot database before running the rest

-- Sample data for testing (run after starting the server to create tables)

-- Insert sample users
INSERT INTO users (name, phone, email, password, role, created_at, updated_at) VALUES
('Admin User', '9841234567', 'admin@lunaiot.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye7cJNr/PHsgJKSDGOm8hJgxFLGK6tHDu', 0, NOW(), NOW()), -- password: admin123
('John Doe', '9841234568', 'john@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye7cJNr/PHsgJKSDGOm8hJgxFLGK6tHDu', 1, NOW(), NOW()), -- password: admin123
('Jane Smith', '9841234569', 'jane@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMye7cJNr/PHsgJKSDGOm8hJgxFLGK6tHDu', 1, NOW(), NOW()); -- password: admin123

-- Insert sample devices
INSERT INTO devices (imei, sim_no, sim_operator, protocol, created_at, updated_at) VALUES
('123456789012345', '9841234570', 'Ncell', 'GT06', NOW(), NOW()),
('123456789012346', '9841234571', 'Ntc', 'GT06', NOW(), NOW()),
('123456789012347', '9841234572', 'Ncell', 'GT06', NOW(), NOW());

-- Insert sample vehicles
INSERT INTO vehicles (imei, reg_no, name, odometer, mileage, min_fuel, overspeed, vehicle_type, created_at, updated_at) VALUES
('123456789012345', 'BA-1-PA-1234', 'Company Car 1', 15000.00, 12.50, 10.00, 80, 'car', NOW(), NOW()),
('123456789012346', 'BA-2-CHA-5678', 'Delivery Truck', 45000.00, 8.30, 50.00, 60, 'truck', NOW(), NOW()),
('123456789012347', 'BA-3-PA-9012', 'School Bus', 25000.00, 6.20, 80.00, 40, 'school_bus', NOW(), NOW());

-- Insert sample GPS data
INSERT INTO gps_data (imei, timestamp, latitude, longitude, speed, course, satellites, gps_real_time, gps_positioned, ignition, charger, gps_tracking, voltage_level, voltage_status, gsm_signal, gsm_status, protocol_name, raw_packet, created_at, updated_at) VALUES
('123456789012345', NOW() - INTERVAL '1 hour', 27.717245, 85.323959, 45, 180, 8, true, true, 'ON', 'CONNECTED', 'ENABLED', 4, 'MEDIUM', 3, 'GOOD', 'GPS_LBS_STATUS', '787811001234...', NOW(), NOW()),
('123456789012345', NOW() - INTERVAL '30 minutes', 27.720245, 85.326959, 50, 185, 9, true, true, 'ON', 'CONNECTED', 'ENABLED', 4, 'MEDIUM', 4, 'EXCELLENT', 'GPS_LBS_STATUS', '787811001234...', NOW(), NOW()),
('123456789012345', NOW() - INTERVAL '15 minutes', 27.723245, 85.329959, 0, 185, 7, true, true, 'OFF', 'CONNECTED', 'ENABLED', 4, 'MEDIUM', 3, 'GOOD', 'GPS_LBS_STATUS', '787811001234...', NOW(), NOW()),
('123456789012346', NOW() - INTERVAL '45 minutes', 27.715245, 85.321959, 30, 90, 6, true, true, 'ON', 'CONNECTED', 'ENABLED', 3, 'LOW', 2, 'WEAK', 'GPS_LBS_STATUS', '787811001234...', NOW(), NOW()),
('123456789012347', NOW() - INTERVAL '20 minutes', 27.719245, 85.325959, 25, 270, 8, true, true, 'ON', 'CONNECTED', 'ENABLED', 5, 'HIGH', 4, 'EXCELLENT', 'GPS_LBS_STATUS', '787811001234...', NOW(), NOW());

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_gps_data_imei_timestamp ON gps_data(imei, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_gps_data_timestamp ON gps_data(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_devices_imei ON devices(imei);
CREATE INDEX IF NOT EXISTS idx_vehicles_imei ON vehicles(imei);
CREATE INDEX IF NOT EXISTS idx_vehicles_reg_no ON vehicles(reg_no);

-- Display summary
SELECT 'Database setup completed successfully!' as status;
SELECT COUNT(*) as total_users FROM users;
SELECT COUNT(*) as total_devices FROM devices;
SELECT COUNT(*) as total_vehicles FROM vehicles;
SELECT COUNT(*) as total_gps_records FROM gps_data; 