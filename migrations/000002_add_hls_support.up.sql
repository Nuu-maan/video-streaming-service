\"-- Migration: Add HLS support to videos table\"  
  
ALTER TABLE videos  
ADD COLUMN IF NOT EXISTS hls_master_path TEXT,  
ADD COLUMN IF NOT EXISTS hls_ready BOOLEAN DEFAULT false,  
ADD COLUMN IF NOT EXISTS streaming_protocol VARCHAR(20) DEFAULT 'progressive'; 
