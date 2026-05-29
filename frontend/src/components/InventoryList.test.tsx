import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { InventoryList } from '../components/InventoryList';
import * as api from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof api>();
  return {
    ...actual,
    fetchInventory: vi.fn(),
    createReservation: vi.fn(),
    getUserId: () => 'test-user',
  };
});

const item = {
  id: 'item-1',
  name: 'Test Sneakers',
  total_quantity: 5,
  reserved_quantity: 0,
  available_quantity: 5,
};

function renderList() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <InventoryList />
    </QueryClientProvider>,
  );
}

describe('InventoryList', () => {
  beforeEach(() => {
    vi.mocked(api.fetchInventory).mockResolvedValue([item]);
  });

  it('reserve happy path shows success feedback', async () => {
    vi.mocked(api.createReservation).mockResolvedValue({
      id: 'res-1',
      item_id: item.id,
      item_name: item.name,
      quantity: 1,
      status: 'active',
      expires_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
    });

    renderList();
    await waitFor(() => expect(screen.getByText('Test Sneakers')).toBeInTheDocument());

    await userEvent.click(screen.getByTestId(`reserve-${item.id}`));

    await waitFor(() =>
      expect(screen.getByTestId(`feedback-${item.id}`)).toHaveTextContent(
        'Reserved 1 unit(s) successfully.',
      ),
    );
    expect(api.createReservation).toHaveBeenCalledWith(item.id, 1);
  });

  it('shows error when insufficient stock', async () => {
    vi.mocked(api.createReservation).mockRejectedValue(
      new Error('Not enough stock available for this reservation'),
    );

    renderList();
    await waitFor(() => expect(screen.getByText('Test Sneakers')).toBeInTheDocument());

    await userEvent.click(screen.getByTestId(`reserve-${item.id}`));

    await waitFor(() => {
      const feedback = screen.getByTestId(`feedback-${item.id}`);
      expect(feedback).toHaveTextContent('Not enough stock');
      expect(feedback).toHaveClass('error');
    });
  });
});
