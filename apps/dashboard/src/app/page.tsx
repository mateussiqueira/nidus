import { readFileSync } from "fs"
import { join } from "path"

export default async function Home() {
  let html = ""
  try {
    html = readFileSync(join(process.cwd(), "public", "index.html"), "utf-8")
  } catch {}
  
  if (!html) return <div>StackRun — Self-hosted PaaS</div>
  
  return <div dangerouslySetInnerHTML={{ __html: html }} />
}
