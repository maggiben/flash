import { useEffect, useState } from 'react';
import { formatCountdown, isExpired, remainingSeconds } from '../utils/reservationTimer';
import type { Reservation } from '../api/client';

type Props = {
  reservation: Reservation;
  onConfirm: (id: string) => void;
  onRelease: (id: string) => void;
  confirmingId: string | null;
  releasingId: string | null;
};

export function ReservationRow({
  reservation,
  onConfirm,
  onRelease,
  confirmingId,
  releasingId,
}: Props) {
  const isConfirmed = reservation.status === 'confirmed';
  const [secondsLeft, setSecondsLeft] = useState(() =>
    remainingSeconds(reservation.expires_at),
  );

  useEffect(() => {
    if (isConfirmed) return;
    const tick = () => setSecondsLeft(remainingSeconds(reservation.expires_at));
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [reservation.expires_at, isConfirmed]);

  const expired = !isConfirmed && isExpired(reservation.expires_at);

  return (
    <li className="reservation-row" data-testid={`reservation-${reservation.id}`}>
      <div>
        <strong>{reservation.item_name}</strong>
        <span className="muted"> × {reservation.quantity}</span>
      </div>
      <div className="reservation-meta">
        {isConfirmed ? (
          <span className="badge badge-confirmed">Confirmed</span>
        ) : expired ? (
          <span className="badge badge-warn">Expired — release to sync</span>
        ) : (
          <span className="badge badge-active">Expires in {formatCountdown(secondsLeft)}</span>
        )}
        {!isConfirmed && (
          <button
            type="button"
            className="btn btn-primary"
            disabled={confirmingId === reservation.id}
            data-testid={`confirm-${reservation.id}`}
            onClick={() => onConfirm(reservation.id)}
          >
            {confirmingId === reservation.id ? 'Confirming…' : 'Confirm'}
          </button>
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
