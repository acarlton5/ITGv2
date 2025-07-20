// Module handling FTL connection logic
pub mod state;
use crate::ftl_codec::{FtlCodec, FtlCommand};
use futures::{SinkExt, StreamExt};
use hex::encode;
use log::{error, info, warn};
use rand::distributions::Uniform;
use rand::{thread_rng, Rng};
use serde_json::{json, Value};
use reqwest::Client;
use tokio_tungstenite::{connect_async, tungstenite::Message};
use state::ConnectionState;
use std::env;
use tokio::net::TcpStream;
use tokio::sync::mpsc;
use tokio_util::codec::Framed;

#[derive(Debug)]
enum FrameCommand {
    Send { data: Vec<String> },
    // Kill,
}
// Represents a client connection
pub struct Connection {}
impl Connection {
    //initialize connection
    pub fn init(stream: TcpStream) {
        //Initialize 2 channels so we can communicate between the frame task and the command handling task
        let (frame_send, mut conn_receive) = mpsc::channel::<FtlCommand>(2);
        let (conn_send, mut frame_receive) = mpsc::channel::<FrameCommand>(2);
        //spawn a task whos sole job is to interact with the frame to send and receive information through the codec
        tokio::spawn(async move {
            let mut frame = Framed::new(stream, FtlCodec::new());
            loop {
                //wait until there is a command present
                match frame.next().await {
                    Some(Ok(command)) => {
                        //send the command to the command handling task
                        match frame_send.send(command).await {
                            Ok(_) => {
                                //wait for the command handling task to send us instructions
                                let command = frame_receive.recv().await;
                                //handle the instructions that we received
                                match handle_frame_command(command, &mut frame).await {
                                    Ok(_) => {}
                                    Err(e) => {
                                        error!("There was an error handing frame command {:?}", e);
                                        return;
                                    }
                                };
                            }
                            Err(e) => {
                                error!(
                                    "There was an error sending the command to the connection Error: {:?}", e
                                );
                                return;
                            }
                        };
                    }
                    Some(Err(e)) => {
                        error!("There was an error {:?}", e);
                        return;
                    }
                    None => {
                        error!("There was a socket reading error");
                        return;
                    }
                };
            }
        });

        tokio::spawn(async move {
            //initialize new connection state
            let mut state = ConnectionState::new();
            loop {
                //wait until the frame task sends us a command
                match conn_receive.recv().await {
                    Some(FtlCommand::Disconnect) => {
                        if let Some(ref key) = state.stream_key {
                            if let Err(e) = notify_stream_end(key).await {
                                error!("Failed to notify end of stream: {:?}", e);
                            }
                        }
                        if let Some(port) = state.rtp_port {
                            free_port(port).await;
                        }
                        return;
                    }
                    //this command is where we tell the client what port to use
                    //WARNING: This command does not work properly.
                    //For some reason the client does not like the port we are sending and defaults to 65535 this is fine for now but will be fixed in the future
                    Some(FtlCommand::Dot) => {
                        if state.rtp_port.is_none() {
                            state.rtp_port = allocate_port().await;
                        }
                        let port = state.rtp_port.unwrap_or(65535);
                        let resp_string = format!("200 hi. Use UDP port {}\n", port);
                        let mut resp = Vec::new();
                        resp.push(resp_string);
                        //tell the frame task to send our response
                        match conn_send.send(FrameCommand::Send { data: resp }).await {
                            Ok(_) => {
                                info!("Client connected!");
                                state.print()
                            }
                            Err(e) => {
                                error!("Error sending to frame task (From: Handle HMAC) {:?}", e);
                                return;
                            }
                        }
                    }
                    Some(command) => {
                        handle_command(command, &conn_send, &mut state).await;
                    }
                    None => {
                        error!("Nothing received from the frame");
                        if let Some(ref key) = state.stream_key {
                            let _ = notify_stream_end(key).await;
                        }
                        if let Some(port) = state.rtp_port {
                            free_port(port).await;
                        }
                        return;
                    }
                }
            }
        });
    }
}

async fn handle_frame_command(
    command: Option<FrameCommand>,
    frame: &mut Framed<TcpStream, FtlCodec>,
) -> Result<(), String> {
    match command {
        Some(FrameCommand::Send { data }) => {
            let mut d: Vec<String> = data.clone();
            d.reverse();
            while !d.is_empty() {
                let item = d.pop().unwrap();
                match frame.send(item.clone()).await {
                    Ok(_) => {}
                    Err(e) => {
                        info!("There was an error {:?}", e);
                        return Err(format!("There was an error {:?}", e));
                    }
                }
            }

            return Ok(());
        }
        // Some(FrameCommand::Kill) => {
        //     info!("TODO: Implement Kill command");
        //     return Ok(());
        // }
        None => {
            info!("Error receiving command from conn");
            return Err("Error receiving command from conn".to_string());
        }
    };
}

async fn handle_command(
    command: FtlCommand,
    sender: &mpsc::Sender<FrameCommand>,
    conn: &mut ConnectionState,
) {
    match command {
        FtlCommand::HMAC => {
            conn.hmac_payload = Some(generate_hmac());
            let resp = vec!["200 ".to_string(), conn.get_payload(), "\n".to_string()];
            match sender.send(FrameCommand::Send { data: resp }).await {
                Ok(_) => {}
                Err(e) => {
                    error!("Error sending to frame task (From: Handle HMAC) {:?}", e);
                }
            }
        }
        FtlCommand::Connect { data } => {
            // make sure we receive a valid channel id and stream key
            match (data.get("stream_key"), data.get("channel_id")) {
                (Some(key), Some(_channel_id)) => {
                    conn.stream_key = Some(key.clone());
                    if let Err(e) = notify_stream_start(key.clone()).await {
                        error!("Auth stream error: {:?}", e);
                    }
                    let resp = vec!["200\n".to_string()];
                    match sender.send(FrameCommand::Send { data: resp }).await {
                        Ok(_) => {}
                        Err(e) => error!(
                            "Error sending to frame task (From: Handle Connection) {:?}",
                            e
                        ),
                    }
                }
                
                (None, _) => {
                    error!("No stream key attached to connect command");
                }
                (_, None) => {
                    error!("No channel id attached to connect command");
                }
            }
        }
        FtlCommand::Attribute { data } => {
            match (data.get("key"), data.get("value")) {
                (Some(key), Some(value)) => {
                    // info!("Key: {:?}, value: {:?}", key, value);
                    match key.as_str() {
                        "ProtocolVersion" => conn.protocol_version = Some(value.to_string()),
                        "VendorName" => conn.vendor_name = Some(value.to_string()),
                        "VendorVersion" => conn.vendor_version = Some(value.to_string()),
                        "Video" => {
                            match value.as_str() {
                                "true" => conn.video = true,
                                "false" => conn.video = false,
                                _ => {
                                    error!("Invalid video value! Atrribute parse failed. Value was: {:?}", value);
                                    return;
                                }
                            }
                        }
                        "VideoCodec" => conn.video_codec = Some(value.to_string()),
                        "VideoHeight" => conn.video_height = Some(value.to_string()),
                        "VideoWidth" => conn.video_width = Some(value.to_string()),
                        "VideoPayloadType" => conn.video_payload_type = Some(value.to_string()),
                        "VideoIngestSSRC" => conn.video_ingest_ssrc = Some(value.to_string()),
                        "Audio" => {
                            match value.as_str() {
                                "true" => conn.audio = true,
                                "false" => conn.audio = false,
                                _ => {
                                    error!("Invalid audio value! Atrribute parse failed. Value was: {:?}", value);
                                    return;
                                }
                            }
                        }
                        "AudioCodec" => conn.audio_codec = Some(value.to_string()),
                        "AudioPayloadType" => conn.audio_payload_type = Some(value.to_string()),
                        "AudioIngestSSRC" => conn.audio_ingest_ssrc = Some(value.to_string()),
                        _ => {
                            error!("Invalid attribute command. Attribute parsing failed. Key was {:?}, Value was {:?}", key, value)
                        }
                    }
                    // No actual response is expected but if we do not respond at all the client
                    // stops sending for some reason.
                    let resp = vec!["".to_string()];
                    match sender.send(FrameCommand::Send { data: resp }).await {
                        Ok(_) => {}
                        Err(e) => error!(
                            "Error sending to frame task (From: Handle Connection) {:?}",
                            e
                        ),
                    }
                }
                (None, Some(_value)) => {}
                (Some(_key), None) => {}
                (None, None) => {}
            }
        }
        FtlCommand::Ping => {
            // info!("Handling PING Command");
            let resp = vec!["201\n".to_string()];
            match sender.send(FrameCommand::Send { data: resp }).await {
                Ok(_) => {}
                Err(e) => error!(
                    "Error sending to frame task (From: Handle Connection) {:?}",
                    e
                ),
            }
        }
        _ => {
            warn!("Command not implemented yet. Tell GRVY to quit his day job");
        }
    }
}

async fn notify_stream_start(stream_key: String) -> Result<(), Box<dyn std::error::Error>> {
    let (ws_stream, _) = connect_async("wss://meow.com/stream/auth").await?;
    let (mut write, mut read) = ws_stream.split();
    let msg = json!({ "stream_key": stream_key });
    write.send(Message::Text(msg.to_string().into())).await?;
    if let Some(msg) = read.next().await {
        if let Ok(Message::Text(t)) = msg {
            info!("Auth status: {}", t);
        }
    }
    Ok(())
}

async fn notify_stream_end(stream_key: &str) -> Result<(), Box<dyn std::error::Error>> {
    let (ws_stream, _) = connect_async("wss://meow.com/stream/auth").await?;
    let (mut write, _read) = ws_stream.split();
    let msg = json!({ "stream_key": stream_key, "status": "ended" });
    write.send(Message::Text(msg.to_string().into())).await?;
    Ok(())
}

async fn allocate_port() -> Option<u16> {
    let base = env::var("WEBRTC_SERVER_URL").unwrap_or_else(|_| "http://localhost:8080".to_string());
    let client = Client::new();
    match client.post(format!("{}/stream", base)).send().await {
        Ok(resp) => match resp.json::<Value>().await {
            Ok(v) => v.get("port").and_then(|p| p.as_u64()).map(|p| p as u16),
            Err(_) => None,
        },
        Err(e) => {
            warn!("Port allocation failed: {:?}", e);
            None
        }
    }
}

async fn free_port(port: u16) {
    let base = env::var("WEBRTC_SERVER_URL").unwrap_or_else(|_| "http://localhost:8080".to_string());
    let client = Client::new();
    let _ = client
        .delete(format!("{}/stream?port={}", base, port))
        .send()
        .await;
}

fn generate_hmac() -> String {
    let dist = Uniform::new(0x00, 0xFF);
    let mut hmac_payload: Vec<u8> = Vec::new();
    let mut rng = thread_rng();
    for _ in 0..128 {
        hmac_payload.push(rng.sample(dist));
    }
    encode(hmac_payload.as_slice())
}
