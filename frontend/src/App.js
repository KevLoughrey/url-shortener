import React, { useState } from "react";
import { Tabs, Tab, Form, Button } from "react-bootstrap";
import axios from "axios";

function App() {
    const [key, setKey] = useState("shorten");
    const [inputURL, setInputURL] = useState("");
    const [result, setResult] = useState("");
    const [shortCode, setShortCode] = useState("");
    const [longURL, setLongURL] = useState("");
    const [clickCount, setClickCount] = useState(null);
    const [error, setError] = useState("");

    const shorten = async () => {
        setError("");
        setResult("");

        if (!inputURL.trim()) {
            setError("Please enter a URL");
            return;
        }

        try {
            const apiBaseUrl = process.env.REACT_APP_API_BASE_URL;
            const res = await axios.post(`${apiBaseUrl}/api/shorten`, {
                long_url: inputURL.trim(),
            });
            setResult(res.data.short_url);
        } catch (err) {
            setError(err.response?.data || "An error occurred");
        }
    };

    const expand = async () => {
        setError("");
        setLongURL("");
        setClickCount(null);

        if (!shortCode.trim()) {
            setError("Please enter a short URL or code");
            return;
        }

        try {
            const baseURL = process.env.REACT_APP_BASE_URL;
            const code = shortCode.replace(`${baseURL}/`, "").trim();

            const apiBaseUrl = process.env.REACT_APP_API_BASE_URL;
            const res = await axios.post(`${apiBaseUrl}/api/expand`, {
                short_code: code,
            });
            setLongURL(res.data.long_url);
            setClickCount(res.data.click_count);
        } catch (err) {
            setError(err.response?.data || "Short URL not found");
        }
    };

    const handleTabSelect = (k) => {
        setKey(k);
        setError("");
        setResult("");
        setLongURL("");
        setClickCount(null);
    };

    return (
        <div className="container mt-4">
            <Tabs activeKey={key} onSelect={handleTabSelect} className="mb-3">
                <Tab eventKey="shorten" title="Shorten URL">
                    <Form.Control
                        type="text"
                        placeholder="Enter URL (e.g., google.com or https://google.com)"
                        value={inputURL}
                        onChange={(e) => setInputURL(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && shorten()}
                    />
                    <Button className="mt-2" onClick={shorten}>
                        Shorten
                    </Button>

                    {error && key === "shorten" && (
                        <p className="mt-2 text-danger">{error}</p>
                    )}

                    {result && (
                        <p className="mt-2 text-success">
                            Shortened URL:{" "}
                            <a
                                href={result}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                {result}
                            </a>
                        </p>
                    )}
                </Tab>

                <Tab eventKey="expand" title="Expand URL">
                    <Form.Control
                        type="text"
                        placeholder="Enter short URL or code"
                        value={shortCode}
                        onChange={(e) => setShortCode(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && expand()}
                    />
                    <Button className="mt-2" onClick={expand}>
                        Expand
                    </Button>

                    {error && key === "expand" && (
                        <p className="mt-2 text-danger">{error}</p>
                    )}

                    {longURL && (
                        <p className="mt-2 text-success">
                            Original URL:{" "}
                            <a
                                href={longURL}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                {longURL}
                            </a>{" "}
                            (clicked {clickCount} times)
                        </p>
                    )}
                </Tab>
            </Tabs>
        </div>
    );
}

export default App;
