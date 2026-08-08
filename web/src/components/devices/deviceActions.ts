import { api } from "../../api";
import type { CardPolicy } from "../../types";

async function ok(p: Promise<unknown>): Promise<{ ok: boolean }> {
  try {
    await p;
    return { ok: true };
  } catch {
    return { ok: false };
  }
}

export function enableVoWiFi(deviceId: string) {
  return ok(api(`/devices/${deviceId}/vowifi`, { method: "PATCH", body: { enabled: true } }));
}
export function disableVoWiFi(deviceId: string) {
  return ok(api(`/devices/${deviceId}/vowifi`, { method: "PATCH", body: { enabled: false } }));
}
export function setFlightMode(deviceId: string, enabled: boolean) {
  return ok(api(`/devices/${deviceId}/flight-mode`, { method: "PATCH", body: { enabled } }));
}
export function getCardPolicy(iccid: string) {
  return api<CardPolicy>(`/cards/${iccid}/policy`);
}
export function putCardPolicy(iccid: string, body: { vowifiEnabled: boolean; airplaneEnabled: boolean }) {
  return ok(api(`/cards/${iccid}/policy`, { method: "PUT", body }));
}
