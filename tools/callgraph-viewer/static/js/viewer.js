// SVG viewer component with pan/zoom
export class Viewer {
    constructor(container) {
        this.container = container;
        this.panZoomInstance = null;
        this.currentGraphId = null;
    }

    // Load and display an SVG graph
    async loadGraph(graphId, filePath) {
        try {
            // Show loading state
            this.container.innerHTML = '<div class="loading">Loading...</div>';

            // Fetch SVG content
            const response = await fetch(filePath);
            if (!response.ok) {
                throw new Error(`Failed to load SVG: ${response.statusText}`);
            }

            const svgText = await response.text();

            // Inject SVG into container
            this.container.innerHTML = svgText;

            // Find SVG element
            const svg = this.container.querySelector('svg');
            if (!svg) {
                throw new Error('No SVG element found in file');
            }

            // Ensure SVG has width and height for proper scaling
            if (!svg.hasAttribute('width')) {
                svg.setAttribute('width', '100%');
            }
            if (!svg.hasAttribute('height')) {
                svg.setAttribute('height', '100%');
            }

            // Initialize pan/zoom
            this.initPanZoom(svg);

            // Update current graph ID
            this.currentGraphId = graphId;

            // Update URL hash
            if (window.location.hash !== `#${graphId}`) {
                window.location.hash = graphId;
            }
        } catch (error) {
            this.showError(error.message);
        }
    }

    // Initialize svg-pan-zoom library
    initPanZoom(svg) {
        // Destroy previous instance
        this.destroyPanZoom();

        // Create new instance
        // Note: svg-pan-zoom is loaded from vendor/svg-pan-zoom.min.js
        if (typeof svgPanZoom !== 'undefined') {
            this.panZoomInstance = svgPanZoom(svg, {
                zoomEnabled: true,
                controlIconsEnabled: false, // We have custom controls
                fit: true,
                center: true,
                minZoom: 0.1,
                maxZoom: 20,
                zoomScaleSensitivity: 0.3,
            });

            // Attach custom controls
            this.attachControls();
        } else {
            console.warn('svg-pan-zoom library not loaded');
        }
    }

    // Destroy pan/zoom instance
    destroyPanZoom() {
        if (this.panZoomInstance) {
            this.panZoomInstance.destroy();
            this.panZoomInstance = null;
        }
    }

    // Attach custom zoom controls
    attachControls() {
        const zoomIn = document.getElementById('zoom-in');
        const zoomOut = document.getElementById('zoom-out');
        const resetZoom = document.getElementById('reset-zoom');

        if (zoomIn && this.panZoomInstance) {
            zoomIn.onclick = () => this.panZoomInstance.zoomIn();
        }

        if (zoomOut && this.panZoomInstance) {
            zoomOut.onclick = () => this.panZoomInstance.zoomOut();
        }

        if (resetZoom && this.panZoomInstance) {
            resetZoom.onclick = () => {
                this.panZoomInstance.resetZoom();
                this.panZoomInstance.center();
                this.panZoomInstance.fit();
            };
        }
    }

    // Show error message
    showError(message) {
        this.container.innerHTML = `
            <div class="empty-state">
                <p>Error: ${message}</p>
            </div>
        `;
    }
}
