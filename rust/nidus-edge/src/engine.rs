use crate::EdgeState;
use anyhow::Result;
use std::fs;
use tracing::info;
use wasmtime::{Linker, Module};
use wasmtime_wasi::preview1::{WasiP1Ctx, add_to_linker_async};

pub async fn load_modules(state: &EdgeState) -> Result<()> {
    if !state.wasm_dir.exists() {
        fs::create_dir_all(&state.wasm_dir)?;
        info!("Created WASM directory: {:?}", state.wasm_dir);
        return Ok(());
    }

    let entries = fs::read_dir(&state.wasm_dir)?;
    let mut count = 0;

    let mut linker: Linker<WasiP1Ctx> = Linker::new(&state.engine);
    add_to_linker_async(&mut linker, |t| t)?;

    for entry in entries {
        let entry = entry?;
        let path = entry.path();
        if path.extension().map_or(false, |ext| ext == "wasm") {
            let name = path.file_stem().unwrap().to_string_lossy().to_string();
            let bytes = fs::read(&path)?;

            let module = Module::from_binary(&state.engine, &bytes)?;
            let instance_pre = linker.instantiate_pre(&module)?;
            state.instance_cache.write().put(name.clone(), instance_pre);
            count += 1;
            info!("Loaded WASM module: {} ({:.1}KB)", name, bytes.len() as f64 / 1024.0);
        }
    }

    info!("WASM instance cache: {} modules pre-linked", count);
    Ok(())
}
