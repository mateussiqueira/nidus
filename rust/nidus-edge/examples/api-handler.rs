// StackRun Edge Function — JSON API Handler

use std::io::{self, Write};

fn main() {
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs();
    
    let json = format!(
        r#"{{"status":"ok","runtime":"stackrun-edge-wasm","version":"0.1.0","timestamp":{}}}"#,
        now
    );
    
    println!("Content-Type: application/json\r");
    println!("X-Runtime: stackrun-edge-wasm\r\n\r");
    println!("{}", json);
}
