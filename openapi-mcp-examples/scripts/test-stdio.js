const { spawn, execSync } = require('child_process');

// Install dependencies
try {
  console.log('Installing dependencies...');
  execSync('npm install', { cwd: 'generated-server', stdio: 'inherit' });
  console.log('Dependencies installed.');
} catch (error) {
  console.error(`Error installing dependencies: ${error}`);
  process.exit(1);
}

const server = spawn('node', ['generated-server/index.js', '--transport', 'stdio']);

const request = {
  tool: 'getWidget',
  inputs: {
    // Missing required 'id' parameter
  }
};

server.stdout.on('data', (data) => {
  console.log(`Server response: ${data}`);
  server.kill();
});

server.stderr.on('data', (data) => {
  console.error(`Server error: ${data}`);
  server.kill();
});

server.on('close', (code) => {
  console.log(`Server process exited with code ${code}`);
});

server.stdin.write(JSON.stringify(request));
server.stdin.end();
