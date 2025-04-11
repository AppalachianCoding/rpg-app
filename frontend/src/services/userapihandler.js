import axios from "axios";

const API_BASE_URL = "https://api.dnd-unsecured-stage.appalachiancoding.org";

const classes = {
    async getClassList() {
        try {
            const response = await axios.get(`${API_BASE_URL}/classes`)
            return response.data
        }
        catch (err) {
            console.error("Error fetching classes:", err)
            throw err;
        }
    },

};

export default classes