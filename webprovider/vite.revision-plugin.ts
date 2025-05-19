import { Plugin, UserConfig } from "vite";
import { execSync } from "child_process";

// inspired by npm package 'vite-plugin-git-revision' but reimplemented from
// scratch to simplify and fit in a single file

// execute a command but silently return undefined on errors
function tryExec(command: string): string | undefined {
  try {
    return execSync(command, {
      encoding: "utf8",
      stdio: [ "ignore", "pipe", "ignore" ],
      timeout: 10,
    }).trim();
  } catch {
    return undefined;
  };
};

// try to get the version information with the first successful method
let version = tryExec("git describe --always --long --dirty");
if (version === undefined)
  version = tryExec("bash ../version.sh version");


export function gitversion(): Plugin {
  return {
    name: "git-revision-plugin",
    config: () => <UserConfig>{
      define: {
        VERSION: JSON.stringify(version || "unknown"),
      }
    },
  };
};
