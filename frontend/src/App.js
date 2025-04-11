import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import UserPage from './pages/UserPage'; // Create this page
import AdminPage from './pages/AdminPage'; // Create this page
import './App.css'; // Keep main styles

function App() {
    return (
        <Router>
            <div className="AppContainer"> {/* Optional: A wrapper container */}
                <nav className="main-nav">
                    <ul>
                        <li><Link to="/">Today's Verse</Link></li>
                        <li><Link to="/admin">Admin</Link></li>
                    </ul>
                </nav>

                <main>
                    <Routes>
                        <Route path="/" element={<UserPage />} />
                        <Route path="/admin" element={<AdminPage />} />
                        {/* Optional: Add a 404 Not Found route */}
                        {/* <Route path="*" element={<NotFoundPage />} /> */}
                    </Routes>
                </main>

                {/* Optional: Footer */}
                {/* <footer> <p>© 2023 Bible App</p> </footer> */}
            </div>
        </Router>
    );
}

export default App;
