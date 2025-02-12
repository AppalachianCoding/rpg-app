import { BrowserRouter as Router, Routes, Route} from "react-router-dom"
import Home from "./pages/home";
import Navbar from "./componet/navbar"
import './App.css'

function App() {

  return (

        <Router>
          <div>
            <Navbar />
          </div>
          <Routes>
            <Route path="/" element={<Home />} />
          </Routes>
        </Router>

  )
}

export default App
