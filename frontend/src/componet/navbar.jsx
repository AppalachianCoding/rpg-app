import "../styles/navbar.css";
import { Link } from "react-router-dom";

function Navbar() {
    return (
        <nav className="navbar">
            <div className="log">DnD</div>
            <ul className="nav-links">
                <li><Link to={"/"}>Home</Link></li>
                <li><Link to={"/Characters"}>Characters</Link></li>
            </ul>
        </nav>
    )
}

export default Navbar