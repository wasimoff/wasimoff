#!/usr/bin/env -S deno run --allow-run

// possible compilation targets
const targets = [
  "x86_64-unknown-linux-gnu",
  "aarch64-unknown-linux-gnu",
  "x86_64-pc-windows-msvc",
  "x86_64-apple-darwin",
  "aarch64-apple-darwin",
] as const;
type Target = (typeof targets)[number];

// func to compile for a single target
async function compile(target: Target = Deno.build.target as Target, alias?: string) {
  const cmd = new Deno.Command("deno", {
    args: [
      "compile",
      "--unstable-sloppy-imports",
      "--no-check",
      "--allow-env",
      "--allow-net",
      "--allow-read",
      "--allow-write",
      "--target",
      target,
      "--output",
      `wasimoff-provider-${alias || target}${target.includes("windows") ? ".exe" : ""}`,
      "main.ts",
    ],
  });
  // execute it, inheriting stdio
  return await cmd.spawn().status;
}

// check if arguments exist
if (Deno.args.length === 0) {
  console.warn("pick a target: {linux,windows,darwin}/{amd64,arm64}, native or all");
  Deno.exit(1);
}

// compile all known targets
if (Deno.args.includes("all")) {
  for (const target of targets) {
    console.log(`--> compile ${target}`);
    await compile(target);
  }
} else {
  // or one target per argument
  for (const arg of Deno.args) {
    const alias = arg.replace("/", "-");
    switch (arg) {
      case "native":
        await compile();
        break;
      case "all":
        // handled above, ignore
        break;
      case "linux/amd64":
        await compile("x86_64-unknown-linux-gnu", alias);
        break;
      case "linux/arm64":
        await compile("aarch64-unknown-linux-gnu", alias);
        break;
      case "windows/amd64":
        await compile("x86_64-pc-windows-msvc", alias);
        break;
      case "darwin/amd64":
        await compile("x86_64-apple-darwin", alias);
        break;
      case "darwin/arm64":
        await compile("aarch64-apple-darwin", alias);
        break;
      default:
        // try our luck directly
        await compile(arg as Target);
        break;
    }
  }
}
