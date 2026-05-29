import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { fetchReservations, releaseReservation } from '../api/client';
import { ReservationRow } from './ReservationRow';

const POLL_MS = 3000;

export function ReservationPanel() {
  const queryClient = useQueryClient();
  const [message, setMessage] = useState({ text: '', isError: false });

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['reservations'],
    queryFn: fetchReservations,
    refetchInterval: POLL_MS,
  });

  const release = useMutation({
    mutationFn: releaseReservation,
    onSuccess: () => {
      setMessage({ text: 'Reservation released.', isError: false });
      void queryClient.invalidateQueries({ queryKey: ['reservations'] });
      void queryClient.invalidateQueries({ queryKey: ['inventory'] });
    },
    onError: (err: Error) => setMessage({ text: err.message, isError: true }),
  });

  return (
    <section className="panel" aria-labelledby="reservations-heading">
      <h2 id="reservations-heading">Your active reservations</h2>
      {isLoading && <p className="status">Loading reservations…</p>}
      {isError && <p className="status error" role="alert">{(error as Error).message}</p>}
      {message.text && (
        <p className={`status ${message.isError ? 'error' : 'success'}`} role="status">
          {message.text}
        </p>
      )}
      {!isLoading && !isError && (
        <ul className="reservation-list">
          {(data ?? []).length === 0 ? (
            <li className="empty">No active reservations</li>
          ) : (
            data!.map((r) => (
              <ReservationRow
                key={r.id}
                reservation={r}
                onRelease={(id) => release.mutate(id)}
                releasingId={release.isPending ? (release.variables ?? null) : null}
              />
            ))
          )}
        </ul>
      )}
    </section>
  );
}
