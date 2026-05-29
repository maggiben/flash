import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createReservation, fetchInventory, type InventoryItem } from '../api/client';

const POLL_MS = 3000;

export function InventoryList() {
  const queryClient = useQueryClient();
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [feedback, setFeedback] = useState<{ itemId?: string; text: string; isError: boolean }>({
    text: '',
    isError: false,
  });

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['inventory'],
    queryFn: fetchInventory,
    refetchInterval: POLL_MS,
  });

  const reserve = useMutation({
    mutationFn: ({ itemId, quantity }: { itemId: string; quantity: number }) =>
      createReservation(itemId, quantity),
    onSuccess: (_res, vars) => {
      setFeedback({
        itemId: vars.itemId,
        text: `Reserved ${vars.quantity} unit(s) successfully.`,
        isError: false,
      });
      void queryClient.invalidateQueries({ queryKey: ['inventory'] });
      void queryClient.invalidateQueries({ queryKey: ['reservations'] });
    },
    onError: (err: Error, vars) => {
      setFeedback({ itemId: vars.itemId, text: err.message, isError: true });
    },
  });

  const getQty = (item: InventoryItem) => quantities[item.id] ?? 1;

  if (isLoading) {
    return <p className="status" data-testid="inventory-loading">Loading inventory…</p>;
  }

  if (isError) {
    return (
      <p className="status error" role="alert" data-testid="inventory-error">
        {(error as Error).message}
      </p>
    );
  }

  return (
    <section className="panel" aria-labelledby="inventory-heading">
      <h2 id="inventory-heading">Flash sale inventory</h2>
      <ul className="inventory-list">
        {(data ?? []).map((item) => {
          const qty = getQty(item);
          const isSubmitting = reserve.isPending && reserve.variables?.itemId === item.id;
          const itemFeedback =
            feedback.itemId === item.id && feedback.text ? feedback : null;

          return (
            <li key={item.id} className="inventory-card" data-testid={`item-${item.id}`}>
              <header>
                <h3>{item.name}</h3>
                <span className={`badge ${item.available_quantity === 0 ? 'badge-sold' : 'badge-stock'}`}>
                  {item.available_quantity === 0 ? 'Sold out' : `${item.available_quantity} left`}
                </span>
              </header>
              <dl className="stock-stats">
                <div>
                  <dt>Total</dt>
                  <dd>{item.total_quantity}</dd>
                </div>
                <div>
                  <dt>Reserved</dt>
                  <dd>{item.reserved_quantity}</dd>
                </div>
                <div>
                  <dt>Available</dt>
                  <dd data-testid={`available-${item.id}`}>{item.available_quantity}</dd>
                </div>
              </dl>
              <div className="reserve-row">
                <label htmlFor={`qty-${item.id}`}>Qty</label>
                <input
                  id={`qty-${item.id}`}
                  type="number"
                  min={1}
                  max={Math.max(1, item.available_quantity)}
                  value={qty}
                  disabled={item.available_quantity === 0}
                  onChange={(e) =>
                    setQuantities((prev) => ({
                      ...prev,
                      [item.id]: Math.max(1, Number(e.target.value) || 1),
                    }))
                  }
                />
                <button
                  type="button"
                  className="btn btn-primary"
                  disabled={item.available_quantity === 0 || isSubmitting}
                  data-testid={`reserve-${item.id}`}
                  onClick={() => reserve.mutate({ itemId: item.id, quantity: qty })}
                >
                  {isSubmitting ? 'Reserving…' : 'Reserve'}
                </button>
              </div>
              {itemFeedback && (
                <p
                  className={`item-feedback ${itemFeedback.isError ? 'error' : 'success'}`}
                  role={itemFeedback.isError ? 'alert' : 'status'}
                  data-testid={`feedback-${item.id}`}
                >
                  {itemFeedback.text}
                </p>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
