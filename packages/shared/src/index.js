"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.detectFramework = detectFramework;
function detectFramework(files) {
    if (files.includes("next.config.mjs") || files.includes("next.config.js"))
        return "nextjs";
    if (files.includes("vite.config.ts") || files.includes("vite.config.js"))
        return "vite";
    if (files.includes("pubspec.yaml"))
        return "vaden";
    if (files.includes("package.json"))
        return "express";
    if (files.includes("requirements.txt") || files.includes("Pipfile"))
        return "flask";
    return "generic";
}
//# sourceMappingURL=index.js.map