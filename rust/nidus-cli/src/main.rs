use clap::Parser;

#[derive(Parser)]
#[command(name = "nidus", version = "0.1.0", about = "Nidus CLI v2.0")]
enum Cli {
    /// Check proxy health
    Health {
        #[arg(long, default_value = "http://localhost:8080")]
        url: String,
    },
    /// List projects
    List,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let cli = Cli::parse();

    match cli {
        Cli::Health { url } => {
            let resp = reqwest::get(format!("{}/health", url)).await?;
            println!("{}", resp.text().await?);
        }
        Cli::List => {
            println!("Project listing available in Phase 2");
        }
    }

    Ok(())
}
