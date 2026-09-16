import { appBase } from './appPath'
import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { HomePage } from './pages/HomePage'
import { PracticePage } from './pages/PracticePage'
import { ResultPage } from './pages/ResultPage'
import { TypeStartPage } from './pages/TypeStartPage'

function createQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { staleTime: 30_000, retry: 1 } } })
}

export function AppRoutes() {
  const [queryClient] = useState(createQueryClient)
  return (
    <QueryClientProvider client={queryClient}>
      <Routes>
        <Route element={<Shell />}>
          <Route index element={<HomePage />} />
          <Route path="practice/type/:code" element={<TypeStartPage />} />
          <Route path="practice/:planId/result" element={<ResultPage />} />
          <Route path="practice/:planId/:n?" element={<PracticePage />} />
        </Route>
      </Routes>
    </QueryClientProvider>
  )
}

export default function App() {
  return <BrowserRouter basename={appBase() || "/"}><AppRoutes /></BrowserRouter>
}
