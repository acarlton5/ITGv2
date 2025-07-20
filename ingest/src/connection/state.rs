use log::{info, warn};

/// ConnectionState tracks the attributes negotiated with the FTL client.
#[derive(Debug)]
pub struct ConnectionState {
    pub hmac_payload: Option<String>,
    pub protocol_version: Option<String>,
    pub vendor_name: Option<String>,
    pub vendor_version: Option<String>,
    pub video: bool,
    pub video_codec: Option<String>,
    pub video_height: Option<String>,
    pub video_width: Option<String>,
    pub video_payload_type: Option<String>,
    pub video_ingest_ssrc: Option<String>,
    pub audio: bool,
    pub audio_codec: Option<String>,
    pub audio_payload_type: Option<String>,
    pub audio_ingest_ssrc: Option<String>,
}

impl ConnectionState {
    /// new creates a default connection state.
    pub fn new() -> ConnectionState {
        ConnectionState {
            hmac_payload: None,
            protocol_version: None,
            vendor_name: None,
            vendor_version: None,
            video: false,
            video_codec: None,
            video_height: None,
            video_width: None,
            video_payload_type: None,
            video_ingest_ssrc: None,
            audio: false,
            audio_codec: None,
            audio_ingest_ssrc: None,
            audio_payload_type: None,
        }
    }

    /// get_payload returns the current HMAC payload as a string.
    pub fn get_payload(&self) -> String {
        match &self.hmac_payload {
            Some(payload) => payload.clone(),
            None => String::new(),
        }
    }

    /// print outputs negotiated parameters to the logger.
    pub fn print(&self) {
        match &self.protocol_version {
            Some(p) => info!("Protocol Version: {}", p),
            None => warn!("Protocol Version: None"),
        }
        match &self.vendor_name {
            Some(v) => info!("Vendor Name: {}", v),
            None => warn!("Vendor Name: None"),
        }
        match &self.vendor_version {
            Some(v) => info!("Vendor Version: {}", v),
            None => warn!("Vendor Version: None"),
        }
        match &self.video_codec {
            Some(v) => info!("Video Codec: {}", v),
            None => warn!("Video Codec: None"),
        }
        match &self.video_height {
            Some(v) => info!("Video Height: {}", v),
            None => warn!("Video Height: None"),
        }
        match &self.video_width {
            Some(v) => info!("Video Width: {}", v),
            None => warn!("Video Width: None"),
        }
        match &self.audio_codec {
            Some(a) => info!("Audio Codec: {}", a),
            None => warn!("Audio Codec: None"),
        }
    }
}
