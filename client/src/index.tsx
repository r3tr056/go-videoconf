import React from 'react';
import ReactDOM from 'react-dom/client';
import '@radix-ui/colors/gray-dark.css';
import '@radix-ui/colors/teal-dark.css';
import '@radix-ui/colors/red-dark.css';
import '@radix-ui/colors/amber-dark.css';
import '@radix-ui/colors/green-dark.css';
import './styles/tokens.css';
import './styles/app.css';
import App from './App';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);