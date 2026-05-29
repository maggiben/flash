import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { InventoryList } from './components/InventoryList';
import { ReservationPanel } from './components/ReservationPanel';
import './styles.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 1000 },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="app">
        <header className="app-header">
          <h1>Flash Sale Reservations</h1>
          <p className="subtitle">Hold items for 60 seconds — first come, first served</p>
        </header>
        <main className="layout">
          <InventoryList />
          <ReservationPanel />
        </main>
      </div>
    </QueryClientProvider>
  );
}
