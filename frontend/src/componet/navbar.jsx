import "../styles/navbar.css";
import { Link } from "react-router-dom";

function Navbar() {
    return (
        <nav className="navbar">
            <div className="logo"><Link to={"/"}>DnD</Link></div>
            <ul className="nav-links">
                <li><Link to={"/"}>Home</Link></li>
                <li><Link to={"/charactersheet"}>Characters</Link></li>
            </ul>
        </nav>
    )
}

export default Navbar