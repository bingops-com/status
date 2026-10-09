import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import '@fontsource-variable/red-hat-display';
import '@fontsource-variable/red-hat-text';
import './styles.css';
import { App } from './App';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
