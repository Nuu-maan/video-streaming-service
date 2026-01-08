\"-- Rollback: Remove HLS support from videos table\"  
  
ALTER TABLE videos  
DROP COLUMN IF EXISTS streaming_protocol,  
DROP COLUMN IF EXISTS hls_ready,  
DROP COLUMN IF EXISTS hls_master_path; 
