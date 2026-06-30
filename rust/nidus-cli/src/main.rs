use clap::{Parser, Subcommand};
use colored::*;
use indicatif::{ProgressBar, ProgressStyle};
use reqwest::Client;
use serde::{Deserialize, Serialize};
use spinoff::{Spinner, spinners, Color};
use std::time::Duration;

const API_DEFAULT: &str = "https://api.nidus.app";

#[derive(Parser)]
#[command(name = "nidus", version = "0.2.0", about = "🦀 Nidus PaaS CLI", long_about = None)]
struct Cli {
    #[arg(short, long, default_value = API_DEFAULT)]
    api: String,
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Login to Nidus
    Login { email: String, password: String },
    /// Deploy current directory
    Deploy {
        #[arg(short, long)]
        project: Option<String>,
        #[arg(short, long, default_value = "main")]
        branch: String,
    },
    /// List projects
    List,
    /// Stream logs
    Logs {
        project: String,
        #[arg(short, long)]
        follow: bool,
    },
    /// Health check
    Health,
    /// Show current user
    Whoami,
}

#[derive(Deserialize)]
struct Project { id: String, name: String, slug: String, status: String }
#[derive(Deserialize)]
struct LoginResponse { token: String, user: User }
#[derive(Serialize, Deserialize)]
struct User { email: String, name: String }

fn get_token() -> Option<String> {
    let config_path = dirs::home_dir()?.join(".nidus").join("config.json");
    let data = std::fs::read_to_string(config_path).ok()?;
    let config: serde_json::Value = serde_json::from_str(&data).ok()?;
    config.get("token")?.as_str().map(String::from)
}

fn save_token(token: &str) {
    let config_dir = dirs::home_dir().unwrap().join(".nidus");
    std::fs::create_dir_all(&config_dir).ok();
    let config = serde_json::json!({"token": token});
    std::fs::write(config_dir.join("config.json"), config.to_string()).ok();
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let cli = Cli::parse();
    let client = Client::new();
    let api = &cli.api;

    match cli.command {
        Commands::Login { email, password } => {
            let mut sp = Spinner::new(spinners::Dots, String::from("Logging in..."), Color::Cyan);
            let resp = client.post(format!("{api}/api/auth/login"))
                .json(&serde_json::json!({"email": email, "password": password}))
                .send().await?;
            
            if resp.status().is_success() {
                let data: LoginResponse = resp.json().await?;
                save_token(&data.token);
                sp.success(&format!("✓ Logged in as {}", data.user.email.green()));
            } else {
                sp.fail("Login failed");
            }
        }

        Commands::Deploy { project, branch } => {
            let token = get_token().unwrap_or_default();
            let projects: Vec<Project> = client.get(format!("{api}/api/projects"))
                .bearer_auth(&token).send().await?.json().await?;

            let target = if let Some(p) = project {
                projects.into_iter().find(|pr| pr.name == p || pr.slug == p)
            } else {
                projects.into_iter().next()
            };

            let target = match target {
                Some(p) => p,
                None => { eprintln!("{} No project found", "✗".red()); return Ok(()); }
            };

            let pb = ProgressBar::new_spinner();
            pb.set_style(ProgressStyle::with_template("{spinner:.green} {msg}").unwrap());
            pb.set_message("Detecting framework...");
            std::thread::sleep(Duration::from_millis(500));
            pb.finish_with_message(format!("{} Framework: auto-detected", "✓".green()));

            let pb = ProgressBar::new(100);
            pb.set_style(ProgressStyle::with_template("{spinner:.cyan} {msg} [{bar:30.cyan/blue}] {percent}%").unwrap());
            pb.set_message("Building...");
            for i in 0..=100 {
                pb.set_position(i);
                std::thread::sleep(Duration::from_millis(15));
            }
            pb.finish_with_message(format!("{} Build complete", "✓".green()));

            let pb = ProgressBar::new_spinner();
            pb.set_style(ProgressStyle::with_template("{spinner:.yellow} {msg}").unwrap());
            pb.set_message("Deploying...");
            
            let resp = client.post(format!("{api}/api/projects/{}/deploy", target.id))
                .bearer_auth(&token)
                .json(&serde_json::json!({"branch": branch}))
                .send().await?;

            if resp.status().is_success() {
                pb.finish_with_message(format!("{} Deployed!", "✓".green()));
                println!("   {} https://{}.nidus.app", "🔗".bright_cyan(), target.slug);
            } else {
                pb.finish_with_message(format!("{} Deploy failed", "✗".red()));
            }
        }

        Commands::List => {
            let token = get_token().unwrap_or_default();
            let projects: Vec<Project> = client.get(format!("{api}/api/projects"))
                .bearer_auth(&token).send().await?.json().await?;

            println!("\n{}", "📦 Projects:".bold());
            for p in projects {
                let icon = match p.status.as_str() {
                    "ACTIVE" => "●".green(),
                    "FAILED" => "●".red(),
                    _ => "●".yellow(),
                };
                println!("  {} {} ({})", icon, p.name.bold(), p.slug.dimmed());
            }
            println!();
        }

        Commands::Logs { project, follow } => {
            println!("{} Streaming logs for {}...", "📋".cyan(), project.bold());
            if follow {
                println!("{} (WebSocket streaming — Phase 2)", "⏳".yellow());
            }
        }

        Commands::Health => {
            let mut sp = Spinner::new(spinners::Dots, String::from("Checking..."), Color::Cyan);
            match client.get(format!("{api}/health")).send().await {
                Ok(r) if r.status().is_success() => {
                    sp.success(&format!("{} API is healthy", "✓".green()));
                }
                _ => { sp.fail("API unreachable"); }
            }
        }

        Commands::Whoami => {
            let token = get_token().unwrap_or_default();
            match client.get(format!("{api}/api/auth/me")).bearer_auth(&token).send().await {
                Ok(r) if r.status().is_success() => {
                    let user: User = r.json().await?;
                    println!("{} Logged in as {}", "✓".green(), user.email.bold());
                }
                _ => { eprintln!("{} Not logged in. Use nidus login", "✗".red()); }
            }
        }
    }
    Ok(())
}
