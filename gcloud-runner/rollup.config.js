import typescript from "@rollup/plugin-typescript";
import replace from "@rollup/plugin-replace";

// execute a command but silently return undefined on errors
import { execSync } from "child_process";
import { log } from "console";
function tryExec(command) {
  try {
    return execSync(command, {
      encoding: "utf8",
      stdio: [ "ignore", "pipe", "ignore" ],
    }).trim();
  } catch {
    return undefined;
  };
};

// try to get the version information with git
const version = tryExec("git describe --always --long --dirty");

// transform out typescript main to a single es module
export default {
  input: "main.ts",
  output: {
    file: "main.js",
    format: "es",
  },
  onLog(level, log, handler) {
    // keep warnings about unresolved imports
    if (log.code === "UNRESOLVED_IMPORT") return handler("warn", log);
    // turn all other warnings into errors
    if (level === "warn") return handler("error", log);
    // passthrough everything else
    return handler(level, log);
  },
  plugins: [
    typescript(),
    replace({
      preventAssignment: true,
      values: {
        "process.env.VERSION": JSON.stringify(version || "unknown"),
      },
    }),
  ],
  // external: [ 
  //   // /^@bufbuild\/*/,
  //   /^@google-cloud\//,
  //   /^@bufbuild\//,
  //   /^@bjorn3\//,
  //   /^@zip\.js\//,
  //   /^pyodide/,
  //   /^comlink/,
  // ],
};
