// Nidus Edge Function — Hello World
// Compile: rustc --target wasm32-wasi -O hello.rs

use std::io::{self, Write};

fn main() {
    println!("Content-Type: text/plain\r");
    println!("X-Runtime: nidus-edge-wasm\r\n\r");
    println!("Hello from Nidus Edge!");
    println!("Runtime: WASM (wasmtime)");
    println!("Cold start: <1ms");
    println!("Memory: ~2MB per instance");
}
