package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tcmartin/flowrunner/pkg/plugins"
)

// handleListPlugins handles listing all registered plugins.
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	pluginNames := s.pluginRegistry.List()

	pluginsInfo := []plugins.PluginMetadata{}
	for _, name := range pluginNames {
		plugin, err := s.pluginRegistry.Get(name)
		if err != nil {
			// Log the error but continue, as we don't want one bad plugin to break the whole list
			// In a real scenario, you might want more robust error handling or logging here.
			continue
		}

		nodePlugin, ok := plugin.(plugins.NodePlugin)
		if !ok {
			// Log the error if a registered item is not a NodePlugin
			continue
		}

		pluginsInfo = append(pluginsInfo, plugins.PluginMetadata{
			Name:        nodePlugin.Name(),
			Description: nodePlugin.Description(),
			Version:     nodePlugin.Version(),
			// Add other metadata fields if available in NodePlugin interface
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pluginsInfo)
}

// handleGetPlugin handles retrieving details of a specific plugin by name.
func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pluginName := vars["name"]

	plugin, err := s.pluginRegistry.Get(pluginName)
	if err != nil {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	nodePlugin, ok := plugin.(plugins.NodePlugin)
	if !ok {
		http.Error(w, "Registered item is not a valid plugin", http.StatusInternalServerError)
		return
	}

	pluginInfo := plugins.PluginMetadata{
		Name:        nodePlugin.Name(),
		Description: nodePlugin.Description(),
		Version:     nodePlugin.Version(),
		// Add other metadata fields if available
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pluginInfo)
}
