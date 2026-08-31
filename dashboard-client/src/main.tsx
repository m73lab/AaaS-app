import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from './lib/queryClient';
import { ClientSessionProvider } from './context/ClientSessionContext';
import { ToastProvider } from './context/ToastContext';
import { App } from './App';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <ClientSessionProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </ClientSessionProvider>
      </ToastProvider>
    </QueryClientProvider>
  </React.StrictMode>,
);
