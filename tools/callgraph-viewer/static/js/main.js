// Main application entry point
import { Navigation } from './navigation.js';
import { Viewer } from './viewer.js';
import { Search } from './search.js';

class CallgraphApp {
    constructor() {
        this.navigation = null;
        this.viewer = null;
        this.search = null;
    }

    async init() {
        try {
            // Get DOM elements
            const navContainer = document.getElementById('navigation');
            const viewerContainer = document.getElementById('viewer');
            const searchInput = document.getElementById('search');

            if (!navContainer || !viewerContainer || !searchInput) {
                throw new Error('Required DOM elements not found');
            }

            // Initialize viewer
            this.viewer = new Viewer(viewerContainer);

            // Initialize navigation with graph selection callback
            this.navigation = new Navigation(navContainer, (graphId, graphPath) => {
                this.loadGraph(graphId, graphPath);
            });
            this.navigation.init();

            // Initialize search
            this.search = new Search(searchInput, this.navigation);

            // Handle hash changes (for direct linking and browser back/forward)
            window.addEventListener('hashchange', () => {
                this.loadGraphFromHash();
            });

            // Load initial graph from hash or first available
            this.loadGraphFromHash();

            // Add keyboard shortcuts
            this.setupKeyboardShortcuts();
        } catch (error) {
            console.error('Failed to initialize app:', error);
        }
    }

    // Load graph from URL hash
    loadGraphFromHash() {
        const hash = window.location.hash.slice(1); // Remove #

        if (hash) {
            // Find graph link with this ID
            const link = document.querySelector(`[data-graph-id="${hash}"]`);
            if (link) {
                const graphPath = link.dataset.graphPath;
                this.loadGraph(hash, graphPath);
            }
        } else {
            // Load first graph
            const firstLink = document.querySelector('.graph-link');
            if (firstLink) {
                const graphId = firstLink.dataset.graphId;
                const graphPath = firstLink.dataset.graphPath;
                this.loadGraph(graphId, graphPath);
            }
        }
    }

    // Load a graph by ID and path
    async loadGraph(graphId, graphPath) {
        await this.viewer.loadGraph(graphId, graphPath);
        this.navigation.setActive(graphId);
    }

    // Setup keyboard shortcuts
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ignore if typing in search
            if (document.activeElement === document.getElementById('search')) {
                return;
            }

            // / - Focus search
            if (e.key === '/') {
                e.preventDefault();
                document.getElementById('search').focus();
            }

            // Escape - Clear search and blur
            if (e.key === 'Escape') {
                this.search.clear();
                document.getElementById('search').blur();
            }

            // Arrow keys for navigation
            if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
                this.navigateGraphs(e.key === 'ArrowDown' ? 1 : -1);
                e.preventDefault();
            }
        });
    }

    // Navigate between graphs with arrow keys
    navigateGraphs(direction) {
        const links = Array.from(document.querySelectorAll('.graph-link:not([style*="display: none"])'));
        const currentIndex = links.findIndex(link => link.classList.contains('active'));

        if (currentIndex === -1 && links.length > 0) {
            // No active, select first
            links[0].click();
        } else if (links.length > 0) {
            const nextIndex = (currentIndex + direction + links.length) % links.length;
            links[nextIndex].click();
        }
    }
}

// Bootstrap app when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        const app = new CallgraphApp();
        app.init();
    });
} else {
    const app = new CallgraphApp();
    app.init();
}
