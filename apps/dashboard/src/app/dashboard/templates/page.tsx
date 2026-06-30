"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { api } from "@/lib/api"
import { Search, Code, Globe, Database, Box, ArrowRight, X, Loader2, FileText, Plus } from "lucide-react"

const templates = [
	{ id: "nextjs", name: "Next.js", icon: "⚡", description: "React framework with SSR/SSG", tags: ["react", "nextjs"], framework: "nextjs" },
	{ id: "express", name: "Express API", icon: "🚀", description: "Node.js REST API server", tags: ["node", "express"], framework: "express" },
	{ id: "vite", name: "Vite + React", icon: "💨", description: "Lightning-fast dev server", tags: ["react", "vite"], framework: "vite" },
	{ id: "static", name: "Static Site", icon: "📄", description: "HTML/CSS/JS sem build", tags: ["html", "static"], framework: "static" },
	{ id: "fastify", name: "Fastify", icon: "🏎️", description: "Fast Node.js web framework", tags: ["node", "fastify"], framework: "fastify" },
	{ id: "nuxt", name: "Nuxt", icon: "💚", description: "Vue.js meta-framework", tags: ["vue", "nuxt"], framework: "nuxt" },
	{ id: "svelte", name: "SvelteKit", icon: "🧡", description: "Svelte full-stack framework", tags: ["svelte", "sveltekit"], framework: "sveltekit" },
	{ id: "astro", name: "Astro", icon: "🌌", description: "Static-first site builder", tags: ["astro", "static"], framework: "astro" },
	{ id: "gin", name: "Gin (Go)", icon: "🐹", description: "Go HTTP web framework", tags: ["go", "gin"], framework: "gin" },
	{ id: "django", name: "Django", icon: "🐍", description: "Python web framework", tags: ["python", "django"], framework: "docker" },
	{ id: "wordpress", name: "WordPress", icon: "📝", description: "CMS mais popular do mundo", tags: ["php", "cms"], framework: "docker" },
	{ id: "n8n", name: "n8n", icon: "🔧", description: "Automação de workflows", tags: ["automation", "n8n"], framework: "docker" },
	{ id: "strapi", name: "Strapi", icon: "📰", description: "Headless CMS", tags: ["cms", "node"], framework: "docker" },
	{ id: "pocketbase", name: "PocketBase", icon: "🗃️", description: "Backend as a service", tags: ["backend", "go"], framework: "docker" },
	{ id: "hono", name: "Hono", icon: "🔥", description: "Ultrafast web framework", tags: ["api", "bun"], framework: "docker" },
	{ id: "laravel", name: "Laravel", icon: "🟧", description: "PHP web framework", tags: ["php", "laravel"], framework: "docker" },
	{ id: "remix", name: "Remix", icon: "⛓️", description: "Full-stack web framework", tags: ["react", "remix"], framework: "docker" },
]

export default function TemplatesPage() {
	const router = useRouter()
	const [search, setSearch] = useState("")
	const [deploying, setDeploying] = useState<string | null>(null)
	const [deployError, setDeployError] = useState<string | null>(null)
	const [modal, setModal] = useState<{ open: boolean; template: typeof templates[0] | null }>({ open: false, template: null })
	const [projectName, setProjectName] = useState("")
	const [customModal, setCustomModal] = useState(false)
	const [customYaml, setCustomYaml] = useState("")
	const [customName, setCustomName] = useState("")

	const filtered = templates.filter(
		(t) =>
			t.name.toLowerCase().includes(search.toLowerCase()) ||
			t.description.toLowerCase().includes(search.toLowerCase()) ||
			t.tags.some((tag) => tag.toLowerCase().includes(search.toLowerCase()))
	)

	function openDeployModal(template: typeof templates[0]) {
		setModal({ open: true, template })
		setProjectName(template.id + "-" + Date.now().toString(36))
		setDeployError(null)
	}

	async function handleDeploy() {
		if (!modal.template) return
		setDeploying(modal.template.id)
		setDeployError(null)

		try {
			await api.request("/api/projects", {
				method: "POST",
				body: JSON.stringify({ name: projectName || modal.template.name, framework: modal.template.framework }),
			})
			setModal({ open: false, template: null })
			router.push("/dashboard/projects")
		} catch (err: any) {
			setDeployError(err.message || "Erro ao criar projeto")
		} finally {
			setDeploying(null)
		}
	}

	async function handleCustomDeploy() {
		setDeploying("custom")
		setDeployError(null)

		try {
			const parsed = parseYaml(customYaml)
			await api.request("/api/projects", {
				method: "POST",
				body: JSON.stringify({
					name: customName || parsed.name || "custom-template",
					framework: parsed.framework || "docker",
					templateYaml: customYaml,
					...parsed,
				}),
			})
			setCustomModal(false)
			setCustomYaml("")
			setCustomName("")
			router.push("/dashboard/projects")
		} catch (err: any) {
			setDeployError(err.message || "Erro ao criar projeto com template customizado")
		} finally {
			setDeploying(null)
		}
	}

	function parseYaml(yaml: string): Record<string, any> {
		const result: Record<string, any> = {}
		let currentKey = ""
		let inList = false
		let listValues: string[] = []
		let inMap = false
		let mapKey = ""

		for (const line of yaml.split("\n")) {
			const trimmed = line.trimEnd()
			if (!trimmed || trimmed.startsWith("#")) continue

			if (trimmed.startsWith("  - ")) {
				if (inList) {
					const val = trimmed.slice(4)
					if (val.includes(":") && !inMap) {
						const [subKey, ...subValParts] = val.split(":")
						const subVal = subValParts.join(":").trim()
						if (currentKey && !result[currentKey]) result[currentKey] = []
						if (Array.isArray(result[currentKey])) {
							;(result[currentKey] as any[]).push({ [subKey.trim()]: subVal || "" })
						}
					} else {
						listValues.push(val)
					}
				}
				continue
			}

			if (trimmed.startsWith("  ") && currentKey && !trimmed.startsWith("  - ")) {
				if (trimmed.includes(":") && !inMap) {
					inList = false
					const [subKey, ...subValParts] = trimmed.trimStart().split(":")
					const subVal = subValParts.join(":").trim()
					if (!result[currentKey]) result[currentKey] = {}
					if (typeof result[currentKey] === "object" && !Array.isArray(result[currentKey])) {
						result[currentKey][subKey.trim()] = subVal || ""
					}
				}
				continue
			}

			if (listValues.length > 0) {
				if (currentKey) result[currentKey] = listValues
				listValues = []
			}

			if (trimmed.includes(":")) {
				const colonIdx = trimmed.indexOf(":")
				const key = trimmed.slice(0, colonIdx).trim()
				const val = trimmed.slice(colonIdx + 1).trim()

				if (val === "") {
					currentKey = key
					inList = false
					listValues = []
				} else if (val === "[string]") {
					result[key] = []
				} else if (val === "[integer]") {
					result[key] = []
				} else {
					result[key] = val
				}
			}
		}

		return result
	}

	function getFrameworkIcon(framework: string) {
		switch (framework) {
			case "docker": return <Database size={16} />
			case "static": return <Globe size={16} />
			default: return <Code size={16} />
		}
	}

	return (
		<div>
			<div className="flex items-center justify-between mb-8">
				<div>
					<h1 className="text-2xl font-bold tracking-tight">Templates</h1>
					<p className="text-sm text-zinc-400 mt-1">Deploy com 1 clique usando modelos prontos</p>
				</div>
			</div>

			<div className="relative mb-6">
				<Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
				<input
					type="text"
					placeholder="Buscar templates..."
					value={search}
					onChange={(e) => setSearch(e.target.value)}
					className="w-full pl-10 pr-4 py-2.5 bg-zinc-900 border border-zinc-800 rounded-lg text-sm text-white placeholder:text-zinc-600 focus:outline-none focus:border-accent transition"
				/>
			</div>

			<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
				{filtered.map((tpl) => (
					<div key={tpl.id} className="card group hover:border-zinc-700 transition-all flex flex-col">
						<div className="flex items-center gap-3 mb-3">
							<div className="w-12 h-12 rounded-lg bg-zinc-800 flex items-center justify-center text-2xl group-hover:bg-zinc-700 transition shrink-0">
								{tpl.icon}
							</div>
							<div className="min-w-0">
								<h3 className="font-semibold text-sm truncate">{tpl.name}</h3>
								<p className="text-xs text-zinc-400 line-clamp-1">{tpl.description}</p>
							</div>
						</div>

						<div className="flex flex-wrap gap-1.5 mb-4">
							<span className="badge badge-building inline-flex items-center gap-1 text-xs">
								{getFrameworkIcon(tpl.framework)}
								{tpl.framework}
							</span>
							{tpl.tags.map((tag) => (
								<span key={tag} className="text-xs px-2 py-0.5 rounded bg-zinc-800 text-zinc-400">{tag}</span>
							))}
						</div>

						<button
							onClick={() => openDeployModal(tpl)}
							disabled={deploying === tpl.id}
							className="btn btn-primary text-sm w-full mt-auto flex items-center justify-center gap-2"
						>
							{deploying === tpl.id ? (
								<><Loader2 size={14} className="animate-spin" /> Deploying...</>
							) : (
								<><ArrowRight size={14} /> Deploy</>
							)}
						</button>
					</div>
				))}

				<div className="card group hover:border-zinc-700 transition-all flex flex-col border-dashed border-zinc-700">
					<div className="flex items-center gap-3 mb-3">
						<div className="w-12 h-12 rounded-lg bg-zinc-800 flex items-center justify-center text-2xl group-hover:bg-zinc-700 transition shrink-0">
							<Plus size={24} className="text-zinc-500" />
						</div>
						<div className="min-w-0">
							<h3 className="font-semibold text-sm truncate">Custom Template</h3>
							<p className="text-xs text-zinc-400 line-clamp-1">Cole seu YAML de template</p>
						</div>
					</div>

					<div className="flex flex-wrap gap-1.5 mb-4">
						<span className="badge badge-building inline-flex items-center gap-1 text-xs">
							<FileText size={16} />
							yaml
						</span>
						<span className="text-xs px-2 py-0.5 rounded bg-zinc-800 text-zinc-400">custom</span>
					</div>

					<button
						onClick={() => { setCustomModal(true); setDeployError(null); setCustomYaml(""); setCustomName("") }}
						className="btn btn-ghost text-sm w-full mt-auto flex items-center justify-center gap-2 border border-zinc-700 hover:border-accent transition"
					>
						<Plus size={14} />
						Custom Template
					</button>
				</div>
			</div>

			{filtered.length === 0 && (
				<div className="card text-center py-16">
					<Box size={48} className="mx-auto mb-4 text-zinc-600" />
					<h2 className="text-lg font-semibold mb-2">Nenhum template encontrado</h2>
					<p className="text-sm text-zinc-400">Tente outro termo de busca.</p>
				</div>
			)}

			{modal.open && modal.template && (
				<div className="fixed inset-0 z-50 flex items-center justify-center">
					<div className="absolute inset-0 bg-black/60" onClick={() => setModal({ open: false, template: null })} />
					<div className="relative bg-zinc-950 border border-zinc-800 rounded-xl p-6 w-full max-w-md mx-4 shadow-2xl">
						<button
							onClick={() => setModal({ open: false, template: null })}
							className="absolute top-4 right-4 p-1 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-white transition"
						>
							<X size={16} />
						</button>

						<div className="flex items-center gap-3 mb-4">
							<div className="w-10 h-10 rounded-lg bg-zinc-800 flex items-center justify-center text-xl">
								{modal.template.icon}
							</div>
							<div>
								<h2 className="text-lg font-semibold">Deploy: {modal.template.name}</h2>
								<p className="text-xs text-zinc-400">{modal.template.description}</p>
							</div>
						</div>

						<div className="mb-4">
							<label className="block text-sm font-medium text-zinc-400 mb-1">Nome do projeto</label>
							<input
								type="text"
								value={projectName}
								onChange={(e) => setProjectName(e.target.value)}
								className="w-full px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-sm text-white placeholder:text-zinc-600 focus:outline-none focus:border-accent transition"
								placeholder="meu-projeto"
							/>
						</div>

						{deployError && (
							<div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
								{deployError}
							</div>
						)}

						<div className="flex items-center gap-3">
							<button
								onClick={() => setModal({ open: false, template: null })}
								className="btn btn-ghost text-sm flex-1"
							>
								Cancelar
							</button>
							<button
								onClick={handleDeploy}
								disabled={deploying === modal.template.id}
								className="btn btn-primary text-sm flex-1 flex items-center justify-center gap-2"
							>
								{deploying === modal.template.id ? (
									<><Loader2 size={14} className="animate-spin" /> Criando...</>
								) : (
									<><ArrowRight size={14} /> Deploy</>
								)}
							</button>
						</div>
					</div>
				</div>
			)}

			{customModal && (
				<div className="fixed inset-0 z-50 flex items-center justify-center">
					<div className="absolute inset-0 bg-black/60" onClick={() => setCustomModal(false)} />
					<div className="relative bg-zinc-950 border border-zinc-800 rounded-xl p-6 w-full max-w-lg mx-4 shadow-2xl">
						<button
							onClick={() => setCustomModal(false)}
							className="absolute top-4 right-4 p-1 rounded-md hover:bg-zinc-800 text-zinc-500 hover:text-white transition"
						>
							<X size={16} />
						</button>

						<div className="flex items-center gap-3 mb-4">
							<div className="w-10 h-10 rounded-lg bg-zinc-800 flex items-center justify-center text-xl">
								<FileText size={20} className="text-zinc-400" />
							</div>
							<div>
								<h2 className="text-lg font-semibold">Custom Template</h2>
								<p className="text-xs text-zinc-400">Cole o YAML do seu template</p>
							</div>
						</div>

						<div className="mb-4">
							<label className="block text-sm font-medium text-zinc-400 mb-1">Nome do projeto</label>
							<input
								type="text"
								value={customName}
								onChange={(e) => setCustomName(e.target.value)}
								className="w-full px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-sm text-white placeholder:text-zinc-600 focus:outline-none focus:border-accent transition"
								placeholder="meu-projeto"
							/>
						</div>

						<div className="mb-4">
							<label className="block text-sm font-medium text-zinc-400 mb-1">Template YAML</label>
							<textarea
								value={customYaml}
								onChange={(e) => setCustomYaml(e.target.value)}
								rows={12}
								className="w-full px-3 py-2 bg-zinc-900 border border-zinc-800 rounded-lg text-sm text-white placeholder:text-zinc-600 focus:outline-none focus:border-accent transition font-mono resize-none"
								placeholder={`name: meu-template
version: "1.0"
description: descricao curta
icon: "🔧"
category: web
tags:
  - tag1
framework: docker
ports:
  - 3000`
								}
							/>
						</div>

						{deployError && (
							<div className="mb-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
								{deployError}
							</div>
						)}

						<div className="flex items-center gap-3">
							<button
								onClick={() => setCustomModal(false)}
								className="btn btn-ghost text-sm flex-1"
							>
								Cancelar
							</button>
							<button
								onClick={handleCustomDeploy}
								disabled={deploying === "custom" || !customYaml.trim()}
								className="btn btn-primary text-sm flex-1 flex items-center justify-center gap-2"
							>
								{deploying === "custom" ? (
									<><Loader2 size={14} className="animate-spin" /> Criando...</>
								) : (
									<><ArrowRight size={14} /> Deploy</>
								)}
							</button>
						</div>
					</div>
				</div>
			)}
		</div>
	)
}
