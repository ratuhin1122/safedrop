import { apiRequest, jsonBody } from "@/api/client";
import { authSchema, credentialsSchema, registerSchema, type Credentials, type RegisterInput } from "@/api/schemas";
import { setAccessToken } from "@/api/session";

export async function register(input: RegisterInput, signal?: AbortSignal) {
  const result = await apiRequest("/auth/register", authSchema, {
    method: "POST", auth: false, signal, ...jsonBody(registerSchema.parse(input)),
  });
  setAccessToken(result.token);
  return result;
}

export async function login(input: Credentials, signal?: AbortSignal) {
  const result = await apiRequest("/auth/login", authSchema, {
    method: "POST", auth: false, signal, ...jsonBody(credentialsSchema.parse(input)),
  });
  setAccessToken(result.token);
  return result;
}
