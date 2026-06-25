#!/usr/bin/env node

const { program } = require('commander');
const chalk = require('chalk');
const ora = require('ora');
const Conf = require('conf');
const https = require('https');
const http = require('http');

const config = new Conf({ projectName: 'nidus' });

const API_URL = config.get('apiUrl') || 'http://localhost:3001';

function request(method, path, body) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, API_URL);
    const mod = url.protocol === 'https:' ? https : http;
    const token = config.get('token');
    
    const options = {
      hostname: url.hostname,
      port: url.port,
      path: url.pathname,
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    };

    const req = mod.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          resolve(JSON.parse(data));
        } catch {
          resolve(data);
        }
      });
    });

    req.on('error', reject);
    if (body) req.write(JSON.stringify(body));
    req.end();
  });
}

program
  .name('nidus')
  .description('Self-hosted deploy platform')
  .version('1.0.0');

program
  .command('login')
  .description('Login to Nidus')
  .argument('<email>')
  .argument('<password>')
  .action(async (email, password) => {
    const spinner = ora('Logging in...').start();
    try {
      const res = await request('POST', '/api/auth/login', { email, password });
      if (res.token) {
        config.set('token', res.token);
        spinner.succeed('Logged in successfully');
      } else {
        spinner.fail('Login failed');
      }
    } catch (err) {
      spinner.fail(`Login failed: ${err.message}`);
    }
  });

program
  .command('deploy')
  .description('Deploy current directory')
  .option('-b, --branch <branch>', 'Branch to deploy', 'main')
  .action(async (opts) => {
    const spinner = ora('Deploying...').start();
    try {
      const res = await request('POST', '/api/deploy', {
        branch: opts.branch,
        dir: process.cwd(),
      });
      spinner.succeed(`Deployed: ${res.url || res.id}`);
    } catch (err) {
      spinner.fail(`Deploy failed: ${err.message}`);
    }
  });

program
  .command('projects')
  .description('List projects')
  .action(async () => {
    try {
      const res = await request('GET', '/api/projects');
      if (Array.isArray(res)) {
        res.forEach(p => {
          console.log(`${chalk.green('●')} ${p.name} (${p.status})`);
        });
      }
    } catch (err) {
      console.error(chalk.red(`Failed: ${err.message}`));
    }
  });

program.parse();
