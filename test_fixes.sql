-- Test script to validate GPS data issues
-- Check if IMEI 0352312094617803 exists in devices and vehicles tables

-- Check devices table
SELECT 'DEVICES' as table_name, count(*) as count, 
       string_agg(imei, ', ') as imei_list
FROM devices 
WHERE imei = '0352312094617803';

-- Check vehicles table  
SELECT 'VEHICLES' as table_name, count(*) as count,
       string_agg(imei || ' (' || name || ')', ', ') as vehicle_list
FROM vehicles 
WHERE imei = '0352312094617803';

-- Check GPS data table
SELECT 'GPS_DATA' as table_name, 
       count(*) as total_records,
       count(CASE WHEN latitude IS NOT NULL AND longitude IS NOT NULL THEN 1 END) as records_with_coords,
       count(CASE WHEN latitude IS NOT NULL AND longitude IS NOT NULL 
                  AND latitude != 0 AND longitude != 0 THEN 1 END) as records_with_valid_coords,
       max(timestamp) as latest_timestamp
FROM gps_data 
WHERE imei = '0352312094617803';

-- Get latest GPS record for this IMEI (regardless of coordinates)
SELECT 'LATEST_GPS_RECORD' as info,
       imei, latitude, longitude, speed, ignition, timestamp,
       CASE 
         WHEN latitude IS NULL OR longitude IS NULL THEN 'NULL_COORDS'
         WHEN latitude = 0 AND longitude = 0 THEN 'ZERO_COORDS' 
         ELSE 'VALID_COORDS'
       END as coord_status
FROM gps_data 
WHERE imei = '0352312094617803'
ORDER BY timestamp DESC 
LIMIT 1;

-- Check all IMEIs that have GPS data to see which ones work
SELECT 'ALL_IMEIS_WITH_GPS' as info,
       imei, 
       count(*) as total_records,
       count(CASE WHEN latitude IS NOT NULL AND longitude IS NOT NULL 
                  AND latitude != 0 AND longitude != 0 THEN 1 END) as valid_coords,
       max(timestamp) as latest_gps
FROM gps_data 
GROUP BY imei 
ORDER BY latest_gps DESC
LIMIT 5; 