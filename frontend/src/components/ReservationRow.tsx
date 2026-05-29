import { useEffect, useState } from 'react';
import { formatCountdown, isExpired, remainingSeconds } from '../utils/reservationTimer';
import type { Reservation } from '../api/client';

type Props = {
  reservation: Reservation;
  onRelease: (id: string) => void;
  releasingId: string | null;
};

export function ReservationRow({ reservation, onRelease, releasingId }: Props) {
  const [secondsLeft, setSecondsLeft] = useState(() =>
    remainingSeconds(reservation.expires_at),
  );

  useEffect(() => {
    const tick = () => setSecondsLeft(remainingSeconds(reservation.expires_at));
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [reservation.expires_at]);

  const expired = isExpired(reservation.expires_at);

  return (
    <li className="reservation-row" data-testid={`reservation-${reservation.id}`}>
      <div>
        <strong>{reservation.item_name}</strong>
        <span className="muted"> × {reservation.quantity}</span>
      </div>
      <div className="reservation-meta">
        {expired ? (
          <span className="badge badge-warn">Expired — release to sync</span>
        ) : (
          <span className="badge badge-active">Expires in {formatCountdown(secondsLeft)}</span>
        )}
        <button
          type="button"
          className="btn btn-secondary"
          disabled={releasingId === reservation.id}
          onClick={() => onRelease(reservation.id)}
        >
          {releasingId === reservation.id ? 'Releasing…' : 'Release'}
        </button>
      </div>
    </li>
  );
}
