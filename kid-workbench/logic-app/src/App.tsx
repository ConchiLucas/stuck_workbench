import { appBase } from './appPath'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { HomePage } from './pages/HomePage'
import { TypePracticePage } from './pages/TypePracticePage'
import { TypeResultPage } from './pages/TypeResultPage'

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<Shell />}>
        <Route index element={<HomePage />} />
        <Route path="practice/type/:type/result" element={<TypeResultPage />} />
        <Route path="practice/type/:type/:n?" element={<TypePracticePage />} />
      </Route>
    </Routes>
  )
}

export default function App() {
  return <BrowserRouter basename={appBase() || "/"}><AppRoutes /></BrowserRouter>
}
