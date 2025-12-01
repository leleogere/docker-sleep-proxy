// Get check interval from meta tag or default to 5000ms
const checkInterval = parseInt(
    document.querySelector('meta[name="check-interval"]')?.content || '5000'
);

// Get endpoint prefix from meta tag or default to 'sleep-proxy'
const endpointPrefix = document.querySelector('meta[name="endpoint-prefix"]')?.content || 'sleep-proxy';

async function checkHealth() {
    try {
        const response = await fetch(`/${endpointPrefix}/health`);
        const data = await response.json();

        console.log('Health check status:', data.status);

        if (data.status === 'ready') {
            window.location.reload();
        }
    } catch (error) {
        console.log('Still starting...', error);
    }
}

// Start checking immediately
checkHealth();

// Then check every X milliseconds
setInterval(checkHealth, checkInterval);
