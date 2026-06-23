import { Injectable, Logger } from "@nestjs/common"
import { PrismaService } from "../prisma/prisma.service"
import { JwtService } from "@nestjs/jwt"
import * as bcrypt from "bcrypt"

@Injectable()
export class AuthService {
  private readonly logger = new Logger(AuthService.name)

  constructor(
    private readonly prisma: PrismaService,
    private readonly jwt: JwtService,
  ) {}

  async register(email: string, name: string, password: string) {
    const existing = await this.prisma.db.query("SELECT id FROM users WHERE email = $1", [email])
    if (existing.rows.length > 0) throw new Error("Email já cadastrado")

    const hashed = await bcrypt.hash(password, 10)
    const result = await this.prisma.db.query(
      `INSERT INTO users (id, email, name, password, created_at, updated_at)
       VALUES (gen_random_uuid(), $1, $2, $3, NOW(), NOW())
       RETURNING id, name, email`,
      [email, name, hashed],
    )

    const user = result.rows[0]
    return {
      token: this.jwt.sign({ sub: user.id, email: user.email }),
      user: { id: user.id, name: user.name, email: user.email },
    }
  }

  async login(email: string, password: string) {
    const result = await this.prisma.db.query(
      "SELECT id, name, email, password FROM users WHERE email = $1",
      [email],
    )
    if (result.rows.length === 0) throw new Error("Credenciais inválidas")

    const user = result.rows[0]
    const valid = await bcrypt.compare(password, user.password)
    if (!valid) throw new Error("Credenciais inválidas")

    return {
      token: this.jwt.sign({ sub: user.id, email: user.email }),
      user: { id: user.id, name: user.name, email: user.email },
    }
  }

  async me(userId: string) {
    const result = await this.prisma.db.query(
      "SELECT id, name, email, avatar, created_at FROM users WHERE id = $1",
      [userId],
    )
    if (result.rows.length === 0) throw new Error("Usuário não encontrado")
    return result.rows[0]
  }
}
