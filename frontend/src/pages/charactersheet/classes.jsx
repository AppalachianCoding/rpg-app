import classes from "../../tmp";

function Classes() {
    const classesList = [];
    classes.forEach(element => {
        classesList.push(<li>{element}</li>)
    });

    return(
        <div>
            <ul>
               {classesList} 
            </ul>
        </div>
    );
}

export default Classes;