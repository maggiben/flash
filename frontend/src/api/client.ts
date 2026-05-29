export type InventoryItem = {
  id: string;
  name: string;
  total_quantity: number;
  reserved_quantity: number;
  available_quantity: number;
};

export type Reservation = {
  id: string;
  item_id: string;
  item_name: string;
  quantity: number;
  status: 'active' | 'released' | 'expired';
  expires_at: string;
  created_at: string;
};

export type ApiError = {
  error: {
    code: string;
    message: string;
  };
};

const USER_ID_KEY = 'flash-reservation-user-id';

export function getUserId(): string {
  let id = localStorage.getItem(USER_ID_KEY);
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem(USER_ID_KEY, id);
  }
  return id;
}

async function parseError(res: Response): Promise<Error> {
  try {
    const body = (await res.json()) as ApiError;
    return new Error(body.error?.message ?? res.statusText);
  } catch {
    return new Error(res.statusText);
  }
}

export async function fetchInventory(): Promise<InventoryItem[]> {
  const res = await fetch('/api/v1/inventory');
  if (!res.ok) throw await parseError(res);
  const data = (await res.json()) as { items: InventoryItem[] };
  return data.items;
}

export async function fetchReservations(): Promise<Reservation[]> {
  const res = await fetch('/api/v1/reservations', {
    headers: { 'X-User-Id': getUserId() },
  });
  if (!res.ok) throw await parseError(res);
  const data = (await res.json()) as { reservations: Reservation[] };
  return data.reservations;
}

export async function createReservation(itemId: string, quantity: number): Promise<Reservation> {
  const res = await fetch('/api/v1/reservations', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-User-Id': getUserId(),
      'Idempotency-Key': crypto.randomUUID(),
    },
    body: JSON.stringify({ item_id: itemId, quantity }),
  });
  if (!res.ok) throw await parseError(res);
  return (await res.json()) as Reservation;
}

export async function releaseReservation(id: string): Promise<void> {
  const res = await fetch(`/api/v1/reservations/${id}`, {
    method: 'DELETE',
    headers: { 'X-User-Id': getUserId() },
  });
  if (!res.ok) throw await parseError(res);
}
