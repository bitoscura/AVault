function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        var copyButtonIndicator = document.getElementById('copyButtonIndicator');
        copyButtonIndicator.textContent = 'Copied!';
        setTimeout(() => {
            copyButtonIndicator.textContent = '';
        }, 1500);
    }, (err) => {
        alert('Failed to copy text: ' + err);
    });
}

function encryptText() {
    const text = document.getElementById('text').value;
    fetch('/encrypt', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: new URLSearchParams({
            text: text
        })
    })
    .then(response => response.text())
    .then(data => {
        const encryptedText = data.trim(); // Trim any leading/trailing whitespace
        if (encryptedText) {
            document.getElementById('encryptedText').value = encryptedText;
            document.getElementById('copyButton').style.display = 'block';
            document.getElementById('copyButton').setAttribute('onclick', `copyToClipboard('${encryptedText}')`);
        }
    })
    .catch(error => console.error('Error:', error));
}

function clearText() {
    const textArea = document.getElementById('text');
    textArea.value = '';
    const encryptedTextElement = document.getElementById('encryptedText');
    if (encryptedTextElement) {
        encryptedTextElement.textContent = ''; // Clear the encrypted text
        encryptedTextElement.style.display = 'none'; // Hide the encrypted text area
    }
    const copyButton = document.getElementById('copyButton');
    if (copyButton) {
        copyButton.style.display = 'none'; // Hide the copy button
    }
    textArea.focus();
}
