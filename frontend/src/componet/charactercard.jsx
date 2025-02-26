function CharacterCard(props){
    return(
        <div className="character-card">
            <h2>{props.name}</h2>
            <p>Class: {props.classType}</p>
            <p>Level: {props.level}</p>
        </div>
    );
}

export default CharacterCard;