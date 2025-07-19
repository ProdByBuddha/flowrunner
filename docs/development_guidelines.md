# Development Guidelines

## Binary Files Handling

### Overview

Binary files should never be committed to the repository. This includes:
- Compiled executables
- Test binaries
- Generated files
- Large binary assets

### Process for Handling Binaries

1. **Adding New Binaries to .gitignore**

   Whenever you create a new binary file (e.g., through `go build`), immediately add it to the `.gitignore` file:

   ```bash
   echo "/your-binary-name" >> .gitignore
   ```

2. **Naming Conventions for Binaries**

   - Main application binaries should match the project name (e.g., `flowrunner`)
   - CLI tools should have `-cli` suffix (e.g., `flowrunner-cli`)
   - Test binaries should have `test_` prefix (e.g., `test_jwt_auth`)

3. **Removing Accidentally Committed Binaries**

   If you accidentally commit a binary file, remove it from git tracking:

   ```bash
   git rm --cached path/to/binary
   ```

   Then add it to `.gitignore` and commit the changes.

4. **Building Binaries**

   Use the following commands to build binaries:

   ```bash
   # Main application
   go build -o flowrunner ./cmd/flowrunner

   # CLI tool
   go build -o flowrunner-cli ./cmd/flowrunner-cli

   # Test binary
   go build -o test_name ./test_name.go
   ```

5. **Running Tests**

   Test binaries should be built and run separately:

   ```bash
   go build -o test_name ./test_name.go
   ./test_name
   ```

### Current Binary Files

The following binary files are currently excluded in `.gitignore`:

- `/flowrunner` - Main application binary
- `/flowrunner-cli` - CLI tool binary
- `/simple-server` - Simple server test binary
- `/test-server` - Test server binary
- `/test_jwt_auth` - JWT authentication test binary
- `/test_flow_management` - Flow management test binary

When adding new binaries, please follow the naming conventions and add them to `.gitignore`.