import { readFileSync, writeFileSync, readdirSync } from "fs";
import { resolve, extname } from "path";

const srcDir = resolve("dist");
const files = readdirSync(srcDir, { recursive: true }).filter(
  (f) => extname(f) === ".js"
);

for (const file of files) {
  const srcPath = resolve(srcDir, file);
  const destPath = srcPath.replace(/\.js$/, ".cjs");
  let content = readFileSync(srcPath, "utf8");

  content = content
    .replace(
      /\bimport\s+\{([^}]+)\}\s+from\s+['"]([^'"]+)['"]/g,
      (_, names, mod) => {
        const cjsMod = mod.replace(/\.js$/, ".cjs");
        return `const {${names}} = require("${cjsMod}")`;
      }
    )
    .replace(
      /\bimport\s+(\w+)\s+from\s+['"]([^'"]+)['"]/g,
      (_, name, mod) => {
        const cjsMod = mod.replace(/\.js$/, ".cjs");
        return `const ${name} = require("${cjsMod}")`;
      }
    )
    .replace(
      /\bimport\s+['"]([^'"]+)['"]/g,
      (_, mod) => {
        const cjsMod = mod.replace(/\.js$/, ".cjs");
        return `require("${cjsMod}")`;
      }
    )
    .replace(/\bexport\s+\{/g, "module.exports = {")
    .replace(/\bexport\s+default\s+/g, "module.exports = ")
    .replace(/\bexport\s+const\s+/g, "const ")
    .replace(/\bexport\s+function\s+/g, "function ")
    .replace(/\bexport\s+class\s+/g, "class ")
    .replace(/\bexport\s+interface\s+/g, "// @ts-ignore\ninterface ")
    .replace(/\bexport\s+type\s+/g, "// @ts-ignore\ntype ")
    .replace(/\bexport\s+abstract\s+/g, "abstract ");

  writeFileSync(destPath, content);
}

console.log("CJS build complete");
