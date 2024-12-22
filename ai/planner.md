You are tasked with creating a **highly detailed, step-by-step implementation plan** for a software project based on the provided design document. Each step must:
1. Clearly explain **file changes** and their exact purpose.
2. Include **detailed reasoning and instructions** without writing actual code.
3. Specify **testing steps** immediately after introducing a new feature or change.

### General Guidelines:

1. **Fine Granularity**: Divide tasks into small, manageable steps that can be independently verified. Avoid combining unrelated actions in a single step.
2. **File-Level Explanation**: For every added, modified, or deleted file, provide a clear description of:
   - **File contents**: What the file will contain and its intended role.
   - **Purpose**: Why this file or change is necessary.
   - **Dependencies**: Any relationships with other files or modules.
3. **Testing After Every Step**: Specify **how to verify functionality**. Include clear instructions for creating tests and explain **what to validate**.
4. **Reasoning**: Provide a rationale for every action to ensure clarity and context.
5. **No Code**: Do not include raw code. Instead, describe the functionality, structure, and behavior that the code should achieve.

### Format for Each Step:

1. **Step Number**: Sequentially numbered.
2. **File Changes**:
   - **Added**: Files being created, with a description of their structure and purpose.
   - **Modified**: Files being altered, with a summary of changes.
   - **Deleted**: Files being removed, with an explanation for their removal.
   - **Renamed/Moved**: Any renaming or relocating of files, with reasoning.
3. **Description**:
   - **Objective**: A concise explanation of the purpose of this step.
   - **Actions**: Detailed instructions for completing the task, without including code.
   - **Rationale**: Why this step is necessary.
4. **Testing Instructions**: Clear steps to validate the changes or new functionality.

---

### Example:

---

**Step 1: Initialize Project with a Basic Structure**

**File Changes**:
- **Added**:
  - `package.json`: A file specifying project metadata (name, version, description, dependencies, and scripts). Include fields for:
    - The project name and version.
    - Dependencies required for the application (e.g., a web server library such as Express).
    - Scripts for running the application and tests.
  - `.gitignore`: A file listing patterns of files and folders to exclude from version control. Ensure it includes:
    - Temporary build files.
    - Dependencies (e.g., `node_modules`).
    - Environment variable files (e.g., `.env`).
  - `src/index.js`: The entry point for the application, which will initialize and configure the server. Include:
    - The initialization of the web server.
    - A single route to confirm the server runs successfully.
  - `README.md`: A file with a short description of the project, instructions for setting up the development environment, and a summary of what the application does.

**Description**:
- **Objective**: Create a foundational structure for the project to support future development and collaboration.
- **Actions**:
  1. Set up version control using Git and initialize a new repository in the project folder.
  2. Create a `package.json` file with details like the project name, version, dependencies, and scripts for starting the server and running tests.
  3. Create a `.gitignore` file to prevent unnecessary files from being tracked by version control.
  4. Establish a `src` directory containing an `index.js` file, which initializes the web server and provides a minimal endpoint to test functionality.
  5. Add a `README.md` file to document the project setup and provide instructions for new contributors.
- **Rationale**:
  - Version control ensures consistent collaboration and tracking of changes.
  - The `package.json` and `.gitignore` files provide essential configuration for dependency management and version control cleanliness.
  - Including an initial route in `src/index.js` confirms the server is functional and ready for further development.

**Testing Instructions**:
1. Run the command to install dependencies (e.g., `npm install`).
2. Start the server using the script defined in `package.json` (e.g., `npm start`).
3. Visit the endpoint (e.g., `http://localhost:3000`) in a browser or using a tool like `curl` to confirm the server responds successfully.
4. Verify that files like `node_modules` are not tracked in version control by inspecting the `.gitignore` functionality.

---

**Step 2: Add Basic Logging Middleware**

**File Changes**:
- **Added**:
  - `src/middleware/logging.js`: A new file defining a middleware function. This function will:
    - Log incoming HTTP requests, including the request method, URL, and a timestamp.
    - Pass control to the next middleware or route handler.
- **Modified**:
  - `src/index.js`: Update this file to include the new logging middleware. Ensure:
    - The middleware is imported and applied before defining any routes.
    - All incoming requests pass through this middleware for logging.

**Description**:
- **Objective**: Implement logging to monitor HTTP requests for debugging and operational insights.
- **Actions**:
  1. Create a `src/middleware` directory to house reusable middleware.
  2. Add a `logging.js` file in this directory, specifying functionality to log the method, URL, and timestamp for each incoming request.
  3. Modify `src/index.js` to import and apply the logging middleware. Ensure it executes for all requests.
- **Rationale**:
  - Logging is essential for monitoring and debugging during development and production.
  - A modular middleware structure promotes reusability and maintainability.

**Testing Instructions**:
1. Restart the server after adding the middleware.
2. Send test requests to the server using tools like Postman or `curl`.
3. Confirm that each request is logged in the terminal, including the HTTP method, URL, and timestamp.
   - Example log: `[2024-12-22T12:00:00Z] GET /`
4. Verify that the middleware does not interfere with the functionality of existing routes.

