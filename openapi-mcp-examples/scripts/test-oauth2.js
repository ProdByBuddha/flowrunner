const { spawn, execSync } = require('child_process');

async function runTest() { // Wrapped in an async function
  // Unset N8N_API_KEY environment variable
  delete process.env.N8N_API_KEY;

  // Install dependencies for the generated server
  try {
    console.log('Installing dependencies for generated-oauth2-server...');
    execSync('npm install', { cwd: 'generated-oauth2-server', stdio: 'inherit' });
    console.log('Dependencies installed for generated-oauth2-server.');
  } catch (error) {
    console.error(`Error installing dependencies for generated-oauth2-server: ${error}`);
    process.exit(1);
  }

  // Install dependencies for the OAuth2 server
  try {
    console.log('Installing dependencies for OAuth2 server...');
    execSync('npm install express body-parser', { cwd: 'tests/tmp', stdio: 'inherit' });
    console.log('Dependencies installed for OAuth2 server.');
  } catch (error) {
    console.error(`Error installing dependencies for OAuth2 server: ${error}`);
    process.exit(1);
}

  // Start the OAuth2 server
  const oauth2Server = spawn('node', ['tests/tmp/oauth2-server.js']);

  oauth2Server.stdout.on('data', (data) => {
    console.log(`OAuth2 Server: ${data}`);
  });

  oauth2Server.stderr.on('data', (data) => {
    console.error(`OAuth2 Server Error: ${data}`);
  });

  // Add a delay to give the OAuth2 server time to start
  await new Promise(resolve => setTimeout(resolve, 2000));

  // Start the generated MCP server
  const mcpServer = spawn('node', ['generated-oauth2-server/index.js', '--transport', 'stdio']);

  mcpServer.stdout.on('data', (data) => {
    console.log(`MCP Server Response: ${data}`);
    oauth2Server.kill();
    mcpServer.kill();
  });

  mcpServer.stderr.on('data', (data) => {
    console.error(`MCP Server Error: ${data}`);
    oauth2Server.kill();
    mcpServer.kill();
  });

  mcpServer.on('close', (code) => {
    console.log(`MCP Server process exited with code ${code}`);
  });

  // Send a request to the MCP server
  const request = {
    tool: 'getWidgets',
    inputs: {
      clientId: 'test',
      clientSecret: 'secret'
    }
  };

  mcpServer.stdin.write(JSON.stringify(request));
  mcpServer.stdin.end();
}

runTest(); // Call the async function
