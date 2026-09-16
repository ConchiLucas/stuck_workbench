import { appBase } from './appPath'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { HomePage } from './pages/HomePage'
import { MapPage } from './pages/MapPage'
import { ModulePage } from './pages/ModulePage'
import { PracticePage } from './pages/PracticePage'
import { QuestionTypeDetailPage } from './pages/QuestionTypeDetailPage'
import { ResultPage } from './pages/ResultPage'

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } })

export function AppRoutes() {
  return <QueryClientProvider client={queryClient}><Routes><Route element={<Shell />}>
    <Route index element={<HomePage />} />
    <Route path="map" element={<MapPage />} />
    <Route path="types/:typeId" element={<QuestionTypeDetailPage />} />
    <Route path="types/addition-equation/practice" element={<QuestionTypeDetailPage />} />
    <Route path="types/addition-equation/result" element={<Navigate to="/types/addition-equation" replace />} />
    <Route path="practice/type/:type/result" element={<Navigate to="/" replace />} />
    <Route path="practice/type/:type/:n?" element={<QuestionTypeDetailPage />} />
    <Route path="module/:moduleCode" element={<ModulePage />} />
    <Route path="practice/:planId" element={<PracticePage />} />
    <Route path="result/:planId" element={<ResultPage />} />
  </Route></Routes></QueryClientProvider>
}

export default function App() { return <BrowserRouter basename={appBase() || "/"}><AppRoutes /></BrowserRouter> }
