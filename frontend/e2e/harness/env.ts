// Entorno validado del harness (tarea 20): destino de la API y base de pruebas.
// Sin valores por defecto para la base: llega por PMT_TEST_DATABASE_URL.
import crypto from 'node:crypto'
import { assertTestDatabase, assertTestTarget, resolveApiTarget } from '../../config/apiTarget'

export const API_TARGET = assertTestTarget(resolveApiTarget(process.env))
export const DB = assertTestDatabase(process.env.PMT_TEST_DATABASE_URL)
export const DATABASE_URL = process.env.PMT_TEST_DATABASE_URL ?? ''
export const RUN_ID = crypto.randomUUID().replaceAll('-', '').slice(0, 8)
export const TARGET_LINE = `Destino: ${API_TARGET} · base ${DB.database} (usuario ${DB.user})`
